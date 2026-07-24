# yonhemoli-mis

永和护理管理端与小程序 API，后端使用 Go、Gin、GORM 和 MySQL。

## 当前业务边界

管理端 `/api/v1` 仅保留：

1. 内部账户管理：登录、会话、管理员账号、启停和重置密码。
2. 用户管理：列表、导出、封禁和解封。
3. 装修：宣传图 URL、客服信息、公司信息、服务项目。
4. 预约管理：查询、分配顾问、状态流转和历史记录。
5. 常见问题：新增、编辑、排序、发布和下架。
6. 阿姨列表：档案、推荐、排序以及 `DRAFT / COMPLETED` 状态。
7. 阿姨申请：小程序求职申请的联系、核验和状态流转。
8. 客服在线：会话、消息、已读和关闭。

小程序 `/api/mini` 保留：登录、当前用户、装修内容、协议、FAQ、服务项目、阿姨列表与详情、预约提交、求职申请和在线客服。

旧订单、店铺、地址、家属、配菜、支付和旧服务项目接口已退出业务边界。

## 数据库升级

执行前建议备份：

```sh
mysqldump -h 127.0.0.1 -P 3306 -u root -p yonghemolimis \
  > yonghemolimis-before-cleanup.sql
```

已有数据库按以下顺序执行：

```sh
mysql -h 127.0.0.1 -P 3306 -u root -p yonghemolimis < sql/init-schema.sql
mysql -h 127.0.0.1 -P 3306 -u root -p yonghemolimis < sql/cleanup-legacy-tables.sql
mysql -h 127.0.0.1 -P 3306 -u root -p yonghemolimis < sql/init-seed.sql
```

其中 `cleanup-legacy-tables.sql` 会迁移阿姨状态并永久删除旧业务表。全新数据库只需执行：

```sh
make db-init
```

开发/联调测试数据：

```sh
mysql -h 127.0.0.1 -P 3306 -u root -p yonghemolimis \
  < sql/mini-business-test-data.sql
```

默认管理员：`admin / admin123`，首次登录后应立即修改密码。

## 装修接口

- `GET /api/v1/decoration`：读取完整装修配置。
- `GET|PUT /api/v1/decoration/banners`：宣传图 URL。
- `GET|PUT /api/v1/decoration/customer-service`：客服名字、电话和头像。
- `GET|PUT /api/v1/decoration/company`：公司 logo、名字、地址、简介、服务保障和联系电话。
- `/api/v1/decoration/services`：服务项目 CRUD。

## 短信配置

```text
MIS_MINI_SMS_ENDPOINT=https://sms-gateway.example.com/send
MIS_MINI_SMS_TOKEN=your-token
```

仅非 `release` 环境可使用 `MIS_MINI_SMS_TEST_CODE=123456` 联调。

## 抖音小程序登录配置

```text
MIS_MINI_DOUYIN_APPID=ttxxxxxxxxxxxxxxxx
MIS_MINI_DOUYIN_SECRET=从抖音开放平台开发配置获取
MIS_MINI_DOUYIN_PHONE_PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\\n...\\n-----END RSA PRIVATE KEY-----"
```

首次授权会要求用户通过 `getPhoneNumber` 绑定手机号；请先在抖音开放平台为该小程序申请“获取用户手机号”能力，并在“开发配置 > 应用公钥”上传与此私钥匹配的公钥。

## 支付宝小程序登录配置

```text
MIS_MINI_ALIPAY_APPID=2021xxxxxxxxxxxx
MIS_MINI_ALIPAY_APP_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\\n...\\n-----END PRIVATE KEY-----"
```

支付宝端调用 `my.getAuthCode({ scopes: ['auth_base'] })`；服务端用 RSA2 签名请求换取稳定 `user_id`。首次登录无需伪造手机号，手机号应只在用户主动授权后另行绑定。

## 验证

```sh
go test ./...
go vet ./...
```
