package mis

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"yonghemolimis/src/apps/api/response"
	"yonghemolimis/src/dao/db"

	"github.com/gin-gonic/gin"
)

func ListCareServiceCategories(c *gin.Context) {
	rows, err := db.ListMiniServiceCategories(nil)
	if err != nil {
		response.Error(c, 500, "查询服务分类失败")
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, categoryAdminView(row))
	}
	response.OK(c, gin.H{"list": items})
}

func SaveCareServiceCategory(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	var req struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Subtitle    string   `json:"subtitle"`
		Description string   `json:"description"`
		IconURL     string   `json:"iconUrl"`
		Tags        []string `json:"tags"`
		Enabled     *bool    `json:"enabled"`
		Sort        int      `json:"sort"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, 400, "参数错误")
		return
	}
	if id == "" {
		id = strings.TrimSpace(req.ID)
	}
	if id == "" || strings.TrimSpace(req.Name) == "" {
		response.Error(c, 400, "ID 和名称不能为空")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row := &db.MiniServiceCategoryDO{ID: id, Name: strings.TrimSpace(req.Name), Subtitle: strings.TrimSpace(req.Subtitle), Description: strings.TrimSpace(req.Description), IconURL: strings.TrimSpace(req.IconURL), Tags: db.MarshalStringSlice(req.Tags), Enabled: enabled, Sort: req.Sort}
	if err := db.SaveMiniServiceCategory(row); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.OK(c, gin.H{"item": categoryAdminView(*row)})
}

func DeleteCareServiceCategory(c *gin.Context) {
	if err := db.DeleteMiniServiceCategory(c.Param("id")); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.OKMsg(c, "服务分类已删除")
}

func categoryAdminView(row db.MiniServiceCategoryDO) gin.H {
	return gin.H{"id": row.ID, "name": row.Name, "subtitle": row.Subtitle, "description": row.Description, "iconUrl": row.IconURL, "tags": db.UnmarshalStringSlice(row.Tags), "enabled": row.Enabled, "sort": row.Sort, "createdAt": row.CreatedAt, "updatedAt": row.UpdatedAt}
}

func ListCaregivers(c *gin.Context) {
	page, size := misPage(c)
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if status != "" && status != "DRAFT" && status != "COMPLETED" {
		response.Error(c, 400, "status 仅支持 DRAFT 或 COMPLETED")
		return
	}
	rows, total, err := db.ListCaregivers(db.CaregiverListQuery{ServiceID: c.Query("serviceId"), Keyword: c.Query("keyword"), AvailabilityStatus: c.Query("availabilityStatus"), Status: status, Page: page, PageSize: size})
	if err != nil {
		response.Error(c, 500, "查询服务人员失败")
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, caregiverAdminView(row))
	}
	response.OK(c, gin.H{"list": items, "total": total, "page": page, "pageSize": size})
}

func GetCaregiver(c *gin.Context) {
	row, err := db.GetCaregiver(c.Param("id"), false)
	if err != nil {
		response.Error(c, 404, "服务人员不存在")
		return
	}
	response.OK(c, gin.H{"item": caregiverAdminView(*row)})
}

func SaveCaregiver(c *gin.Context) {
	var req struct {
		ID                     string          `json:"id"`
		ContactPhone           string          `json:"contactPhone"`
		AvatarURL              string          `json:"avatarUrl"`
		Name                   string          `json:"name"`
		Age                    int             `json:"age"`
		ExperienceYears        int             `json:"experienceYears"`
		Origin                 string          `json:"origin"`
		ServiceIDs             []string        `json:"serviceIds"`
		Jobs                   []string        `json:"jobs"`
		AvailabilityStatus     string          `json:"availabilityStatus"`
		Rating                 float64         `json:"rating"`
		ServiceCount           int             `json:"serviceCount"`
		Recommended            bool            `json:"recommended"`
		Introduction           string          `json:"introduction"`
		Education              string          `json:"education"`
		Ethnicity              string          `json:"ethnicity"`
		Zodiac                 string          `json:"zodiac"`
		BirthDate              string          `json:"birthDate"`
		Constellation          string          `json:"constellation"`
		Skills                 []string        `json:"skills"`
		Certificates           interface{}     `json:"certificates"`
		IdentityVerified       bool            `json:"identityVerified"`
		PhysicalExamVerified   bool            `json:"physicalExamVerified"`
		MedicalReportImageURLs []string        `json:"medicalReportImageUrls"`
		PersonalInfo           interface{}     `json:"personalInfo"`
		WorkHistory            interface{}     `json:"workHistory"`
		PhotoURLs              []string        `json:"photoUrls"`
		DisplayFields          map[string]bool `json:"displayFields"`
		Status                 string          `json:"status"`
		Source                 string          `json:"source"`
		Sort                   int             `json:"sort"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, 400, "参数错误")
		return
	}
	req.BirthDate = strings.TrimSpace(req.BirthDate)
	if req.BirthDate != "" {
		if _, err := time.Parse("2006-01-02", req.BirthDate); err != nil {
			response.Error(c, 400, "出生日期格式应为 YYYY-MM-DD")
			return
		}
	}
	certificates := db.NormalizeJSONObjectList(req.Certificates, "name")
	workHistory := db.NormalizeJSONObjectList(req.WorkHistory, "role")
	// 联系方式只保留在顶层 ContactPhone（MIS 专用），不允许混入会下发给小程序的 personalInfo。
	personalInfo := db.SanitizeCaregiverPersonalInfo(req.PersonalInfo)
	// 创建档案时由服务端生成 ID 且固定为草稿；发布状态只能通过专用接口变更。
	if id := strings.TrimSpace(c.Param("id")); id != "" {
		req.ID = id
	} else {
		req.ID = ""
	}
	if req.Rating < 0 || req.Rating > 5 || req.Age < 0 || req.Age > 70 {
		response.Error(c, 400, "年龄或评分不符合要求")
		return
	}
	for _, serviceID := range req.ServiceIDs {
		if _, err := db.GetMiniServiceCategory(serviceID, false); err != nil {
			response.Error(c, 400, "服务分类不存在: "+serviceID)
			return
		}
	}
	source := strings.ToUpper(strings.TrimSpace(req.Source))
	if source == "" {
		source = "ADMIN"
	}
	if source != "ADMIN" && source != "SELF_SUBMITTED" {
		response.Error(c, 400, "source 仅支持 ADMIN 或 SELF_SUBMITTED")
		return
	}
	status := "DRAFT"
	applicationID := ""
	if req.ID != "" {
		existing, err := db.GetCaregiver(req.ID, false)
		if err != nil {
			response.Error(c, 404, "服务人员不存在")
			return
		}
		applicationID = existing.ApplicationID
		status = existing.Status
	}
	if status == "COMPLETED" && (strings.TrimSpace(req.Name) == "" || req.Age < 18 || len(req.ServiceIDs) == 0) {
		response.Error(c, 400, "已发布档案必须填写姓名、有效年龄和至少一个服务项目")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "待完善资料"
	}
	row := &db.CaregiverDO{ID: req.ID, ApplicationID: applicationID, ContactPhone: strings.TrimSpace(req.ContactPhone), AvatarURL: req.AvatarURL, Name: strings.TrimSpace(req.Name), Age: req.Age, ExperienceYears: req.ExperienceYears, Origin: req.Origin,
		ServiceIDs: db.MarshalStringSlice(req.ServiceIDs), Jobs: db.MarshalStringSlice(req.Jobs), AvailabilityStatus: req.AvailabilityStatus, Rating: req.Rating, ServiceCount: req.ServiceCount, Recommended: req.Recommended,
		Introduction: req.Introduction, Education: req.Education, Ethnicity: req.Ethnicity, Zodiac: req.Zodiac, BirthDate: req.BirthDate, Constellation: req.Constellation, Skills: db.MarshalStringSlice(req.Skills), Certificates: mustJSON(certificates, "[]"),
		IdentityVerified: req.IdentityVerified, PhysicalExamVerified: req.PhysicalExamVerified, MedicalReportImageURLs: db.MarshalStringSlice(req.MedicalReportImageURLs), PersonalInfo: mustJSON(personalInfo, "{}"),
		WorkHistory: mustJSON(workHistory, "[]"), PhotoURLs: db.MarshalStringSlice(req.PhotoURLs), DisplayFields: mustJSON(req.DisplayFields, "{}"), Status: status, Source: source, Sort: req.Sort}
	if !row.PhysicalExamVerified {
		row.MedicalReportImageURLs = "[]"
	}
	if err := db.SaveCaregiver(row); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.OK(c, gin.H{"item": caregiverAdminView(*row)})
}

func DeleteCaregiver(c *gin.Context) {
	if err := db.DeleteCaregiver(c.Param("id")); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.OKMsg(c, "服务人员已删除")
}

func UpdateCaregiverStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&req) != nil {
		response.Error(c, 400, "参数错误")
		return
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status != "DRAFT" && status != "COMPLETED" {
		response.Error(c, 400, "status 仅支持 DRAFT 或 COMPLETED")
		return
	}
	row, err := db.GetCaregiver(c.Param("id"), false)
	if err != nil {
		response.Error(c, 404, "服务人员不存在")
		return
	}
	if status == "COMPLETED" && (strings.TrimSpace(row.Name) == "" || row.Age < 18 || len(db.UnmarshalStringSlice(row.ServiceIDs)) == 0) {
		response.Error(c, 400, "发布前须填写姓名、有效年龄和至少一个服务项目")
		return
	}
	if err := db.UpdateCaregiverStatus(row.ID, status); err != nil {
		response.Error(c, 500, "更新档案状态失败")
		return
	}
	response.OKMsg(c, "档案状态已更新")
}

func caregiverAdminView(row db.CaregiverDO) gin.H {
	return gin.H{"id": row.ID, "applicationId": row.ApplicationID, "contactPhone": row.ContactPhone, "avatarUrl": row.AvatarURL, "name": row.Name, "age": row.Age, "experienceYears": row.ExperienceYears, "origin": row.Origin, "serviceIds": db.UnmarshalStringSlice(row.ServiceIDs), "jobs": db.UnmarshalStringSlice(row.Jobs), "availabilityStatus": row.AvailabilityStatus, "rating": row.Rating, "serviceCount": row.ServiceCount, "recommended": row.Recommended, "introduction": row.Introduction, "education": row.Education, "ethnicity": row.Ethnicity, "zodiac": row.Zodiac, "birthDate": row.BirthDate, "constellation": row.Constellation, "skills": db.UnmarshalStringSlice(row.Skills), "certificates": db.NormalizeJSONObjectList(jsonValue(row.Certificates, []interface{}{}), "name"), "identityVerified": row.IdentityVerified, "physicalExamVerified": row.PhysicalExamVerified, "medicalReportImageUrls": db.UnmarshalStringSlice(row.MedicalReportImageURLs), "personalInfo": jsonValue(row.PersonalInfo, map[string]interface{}{}), "workHistory": db.NormalizeJSONObjectList(jsonValue(row.WorkHistory, []interface{}{}), "role"), "photoUrls": db.UnmarshalStringSlice(row.PhotoURLs), "displayFields": jsonValue(row.DisplayFields, map[string]bool{}), "status": row.Status, "source": row.Source, "sort": row.Sort, "createdAt": row.CreatedAt, "updatedAt": row.UpdatedAt}
}

func ListDemands(c *gin.Context) {
	page, size := misPage(c)
	rows, total, err := db.ListDemands(db.BusinessListQuery{Status: c.Query("status"), Keyword: c.Query("keyword"), Page: page, PageSize: size})
	if err != nil {
		response.Error(c, 500, "查询服务需求失败")
		return
	}
	response.OK(c, gin.H{"list": rows, "total": total, "page": page, "pageSize": size})
}
func ListResumes(c *gin.Context) {
	page, size := misPage(c)
	rows, total, err := db.ListResumes(db.BusinessListQuery{Status: c.Query("status"), Keyword: c.Query("keyword"), Page: page, PageSize: size})
	if err != nil {
		response.Error(c, 500, "查询简历失败")
		return
	}
	response.OK(c, gin.H{"list": rows, "total": total, "page": page, "pageSize": size})
}

func UpdateDemandStatus(c *gin.Context) { updateBusinessStatus(c, "DEMAND") }
func UpdateResumeStatus(c *gin.Context) { updateBusinessStatus(c, "RESUME") }
func AssignDemand(c *gin.Context)       { assignBusiness(c, "DEMAND") }
func AssignResume(c *gin.Context)       { assignBusiness(c, "RESUME") }
func assignBusiness(c *gin.Context, entityType string) {
	// The assignee is the authenticated Auth subject. Local administrator
	// accounts no longer exist in this application.
	adminID := c.GetUint("userID")
	if adminID == 0 {
		response.Error(c, 401, "未登录")
		return
	}
	if entityType == "DEMAND" {
		row, err := db.AssignDemand(c.Param("id"), adminID)
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		response.OK(c, gin.H{"item": row})
		return
	}
	row, err := db.AssignResume(c.Param("id"), adminID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.OK(c, gin.H{"item": row})
}
func updateBusinessStatus(c *gin.Context, entityType string) {
	var req struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if c.ShouldBindJSON(&req) != nil || strings.TrimSpace(req.Status) == "" {
		response.Error(c, 400, "状态不能为空")
		return
	}
	operatorID, _ := c.Get("userID")
	adminID, _ := operatorID.(uint)
	if entityType == "DEMAND" {
		row, err := db.UpdateDemandStatus(c.Param("id"), req.Status, req.Note, adminID)
		if err != nil {
			response.Error(c, 400, err.Error())
			return
		}
		response.OK(c, gin.H{"item": row})
		return
	}
	row, err := db.UpdateResumeStatus(c.Param("id"), req.Status, req.Note, adminID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.OK(c, gin.H{"item": row})
}

func BusinessStatusHistory(c *gin.Context) {
	entityType := strings.ToUpper(c.Param("entityType"))
	if entityType != "DEMAND" && entityType != "RESUME" {
		response.Error(c, 400, "业务类型无效")
		return
	}
	rows, err := db.ListBusinessStatusHistory(entityType, c.Param("id"))
	if err != nil {
		response.Error(c, 500, "查询状态记录失败")
		return
	}
	response.OK(c, gin.H{"list": rows})
}

func misPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}
func mustJSON(value interface{}, fallback string) string {
	if value == nil {
		return fallback
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fallback
	}
	return string(raw)
}
func jsonValue(raw string, fallback interface{}) interface{} {
	var value interface{}
	if json.Unmarshal([]byte(raw), &value) != nil {
		return fallback
	}
	return value
}
