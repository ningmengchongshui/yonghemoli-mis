package settings

import (
	"bufio"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ServiceName        = "yonghemolimis"
	ServiceDescription = "永和茉莉 系统"
)

var Conf = new(AppConfig)

type AppConfig struct {
	Name       string            `json:"name"`
	Mode       string            `json:"mode"`
	Server     *ServerConfig     `json:"server"`
	Session    *SessionConfig    `json:"session"`
	Log        *LogConfig        `json:"log"`
	DB         *DatabaseConfig   `json:"database"`
	MiniWechat *MiniWechatConfig `json:"mini_wechat"`
	MiniDouyin *MiniDouyinConfig `json:"mini_douyin"`
	MiniAlipay *MiniAlipayConfig `json:"mini_alipay"`
	MiniSMS    *MiniSMSConfig    `json:"mini_sms"`
	OSS        *OSSConfig        `json:"oss"`
	OIDC       *OIDCConfig       `json:"oidc"`
}
type OIDCConfig struct {
	IssuerURL    string `json:"issuer_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURL  string `json:"redirect_url"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type SessionConfig struct {
	ExpiresDays int64 `json:"expires_days"`
}

type LogConfig struct {
	Level    string `json:"level"`
	Filename string `json:"filename"`
}

type DatabaseConfig struct {
	Driver      string `json:"driver"`
	DSN         string `json:"dsn"`
	AutoMigrate bool   `json:"auto_migrate"`
}

type MiniWechatConfig struct {
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

// MiniDouyinConfig 是抖音小程序服务端登录所需的凭据。AppSecret 只能放在
// 服务端环境变量中，绝不能打进小程序包。
type MiniDouyinConfig struct {
	AppID              string `json:"app_id"`
	AppSecret          string `json:"app_secret"`
	PhonePrivateKeyPEM string `json:"phone_private_key_pem"`
}

// MiniAlipayConfig 的私钥只用于服务端 RSA2 签名，严禁下发到小程序。
type MiniAlipayConfig struct {
	AppID         string `json:"app_id"`
	AppPrivateKey string `json:"app_private_key"`
}

// MiniSMSConfig 使用通用 HTTP JSON 网关发送验证码。网关接收
// {"phone":"...","code":"...","scene":"LOGIN"}。
type MiniSMSConfig struct {
	Endpoint string `json:"endpoint"`
	Token    string `json:"token"`
	TestCode string `json:"test_code"`
}

// OSSConfig 与 oss-admin 使用同一套 MinIO/S3 兼容存储配置。
type OSSConfig struct {
	Endpoint      string `json:"endpoint"`
	AccessKey     string `json:"access_key"`
	SecretKey     string `json:"secret_key"`
	UseSSL        bool   `json:"use_ssl"`
	Bucket        string `json:"bucket"`
	PublicBaseURL string `json:"public_base_url"`
}

func Init() error {
	log.Println("=== 永和茉莉 系统配置初始化 ===")
	loadDotEnv(".env")

	Conf.Name = getEnv("MIS_NAME", ServiceName)
	Conf.Mode = getEnv("MIS_MODE", "release")

	Conf.Server = &ServerConfig{
		Host: getEnv("MIS_SERVER_HOST", "127.0.0.1"),
		Port: getEnv("MIS_SERVER_PORT", "8080"),
	}

	Conf.Session = &SessionConfig{
		ExpiresDays: getEnvAsInt64("MIS_SESSION_EXPIRES_DAYS", 30),
	}

	Conf.Log = &LogConfig{
		Level:    getEnv("MIS_LOG_LEVEL", "info"),
		Filename: getEnv("MIS_LOG_FILENAME", "work/logs"),
	}

	// ========== 业务数据库 (MySQL, 读写) ==========
	Conf.DB = &DatabaseConfig{
		Driver:      getEnv("MIS_DB_DRIVER", "mysql"),
		DSN:         getEnv("MIS_DB_DSN", ""),
		AutoMigrate: getEnvAsBool("MIS_DB_AUTO_MIGRATE", false),
	}

	Conf.MiniWechat = &MiniWechatConfig{
		AppID:     getEnv("MIS_MINI_WECHAT_APPID", ""),
		AppSecret: getEnv("MIS_MINI_WECHAT_SECRET", ""),
	}
	Conf.MiniDouyin = &MiniDouyinConfig{
		AppID:              getEnv("MIS_MINI_DOUYIN_APPID", ""),
		AppSecret:          getEnv("MIS_MINI_DOUYIN_SECRET", ""),
		PhonePrivateKeyPEM: strings.ReplaceAll(getEnv("MIS_MINI_DOUYIN_PHONE_PRIVATE_KEY", ""), `\n`, "\n"),
	}
	Conf.MiniAlipay = &MiniAlipayConfig{
		AppID:         getEnv("MIS_MINI_ALIPAY_APPID", ""),
		AppPrivateKey: strings.ReplaceAll(getEnv("MIS_MINI_ALIPAY_APP_PRIVATE_KEY", ""), `\n`, "\n"),
	}
	Conf.MiniSMS = &MiniSMSConfig{
		Endpoint: getEnv("MIS_MINI_SMS_ENDPOINT", ""),
		Token:    getEnv("MIS_MINI_SMS_TOKEN", ""),
		TestCode: getEnv("MIS_MINI_SMS_TEST_CODE", ""),
	}
	Conf.OSS = &OSSConfig{
		Endpoint:      getEnv("MIS_OSS_ENDPOINT", getEnv("MINIO_ENDPOINT", "")),
		AccessKey:     getEnv("MIS_OSS_ACCESS_KEY", getEnv("MINIO_ACCESS_KEY", "")),
		SecretKey:     getEnv("MIS_OSS_SECRET_KEY", getEnv("MINIO_SECRET_KEY", "")),
		UseSSL:        getEnvAsBool("MIS_OSS_USE_SSL", getEnvAsBool("MINIO_USE_SSL", false)),
		Bucket:        getEnv("MIS_OSS_BUCKET", getEnv("MINIO_BUCKET", "oss-ui")),
		PublicBaseURL: getEnv("MIS_OSS_PUBLIC_BASE_URL", getEnv("MINIO_PUBLIC_BASE_URL", "")),
	}
	Conf.OIDC = &OIDCConfig{IssuerURL: getEnv("MIS_OIDC_ISSUER", ""), ClientID: getEnv("MIS_OIDC_CLIENT_ID", ""), ClientSecret: getEnv("MIS_OIDC_CLIENT_SECRET", ""), RedirectURL: getEnv("MIS_OIDC_REDIRECT_URL", "")}

	log.Printf("应用名称: %s", Conf.Name)
	log.Printf("运行模式: %s", Conf.Mode)
	log.Printf("服务地址: http://%s:%s", Conf.Server.Host, Conf.Server.Port)
	log.Printf("业务数据库驱动: %s (MySQL 读写)", Conf.DB.Driver)
	log.Printf("日志级别: %s", Conf.Log.Level)

	log.Println("=== 配置加载完成 ===")
	return nil
}

// loadDotEnv 方便本地直接 go run；线上容器环境变量优先，不会被 .env 覆盖。
func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		log.Printf("环境变量 %s 解析为 int64 失败，使用默认值 %d: %v\n", key, defaultValue, err)
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		log.Printf("环境变量 %s 解析为 bool 失败，使用默认值 %v: %v\n", key, defaultValue, err)
		return defaultValue
	}
	return value
}

var processRunAt string

func GetProcessRunAT() string {
	if processRunAt == "" {
		processRunAt = time.Now().Format("2006-01-02 15:04:05")
	}
	return processRunAt
}

func LogServerInfo() {
	server := Conf.Server
	log.Println("http://" + server.Host + ":" + server.Port)
}

var Version string
var BuildTime string

func SetBaseInfo(version, buildTime string) {
	Version = version
	BuildTime = buildTime
	log.Println("Version: ", Version, "(", BuildTime, ")")
}

type BaseInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
}

func GetBaseInfo() BaseInfo {
	return BaseInfo{
		Version:   Version,
		BuildTime: BuildTime,
	}
}
