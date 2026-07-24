# 永和护理小程序 API

Base URL：`https://yonghemoli.com/api/mini`

成功响应统一为：

```json
{"code":0,"message":"ok","data":{}}
```

登录接口返回 Token，受保护接口使用 `Authorization: Bearer <token>`。

## 登录与用户

- `POST /auth/wechat-login`：微信手机号授权登录。
- `POST /auth/douyin-login`：抖音 `tt.login` 授权登录；首次账号会返回 `needPhoneAuth` 和 10 分钟有效的 `authToken`。
- `POST /auth/douyin-phone-login`：将抖音 `getPhoneNumber` 回调的 `phoneCode`（基础库 3.51+）与 `authToken` 提交，完成手机号绑定并登录；旧版 `encryptedData`、`iv` 仍兼容。
- `POST /auth/alipay-login`：支付宝 `my.getAuthCode({ scopes: ['auth_base'] })` 授权登录。
- `POST /auth/phone-code`：发送短信验证码。
- `POST /auth/phone-login`：手机号验证码登录。
- `GET /users/me`：当前用户，需登录。

## 装修与内容

- `GET /app-config`：宣传图、客服信息、公司信息和服务保障。
- `GET /about`：公司 logo、名字、地址、简介、服务保障和联系电话。
- `GET /agreements/privacy`：隐私政策。
- `GET /agreements/service`：用户服务协议。
- `GET /faqs`：已发布常见问题，可传 `category`。
- `GET /services?enabled=true`：服务项目。

## 阿姨

- `GET /caregivers`：完成状态的阿姨列表。
- `GET /caregivers/{id}`：完成状态的阿姨详情。

列表参数：`serviceId`、`keyword`、`recommended`、`availabilityStatus`、`page`、`pageSize`。

管理端的阿姨状态：

- `DRAFT`：阿姨自行提交或管理员新建后尚未完成资料。
- `COMPLETED`：资料完成，可在小程序公开查询。

数据来源：`SELF_SUBMITTED`、`ADMIN`。

## 预约与求职

- `POST /demands`：提交服务预约/咨询，支持 `Idempotency-Key`。
- `POST /resumes`：提交阿姨求职申请，支持 `Idempotency-Key`。

## 客服在线

- `GET /chat/session`：获取或创建当前用户客服会话，需登录。
- `GET /chat/messages`：客服消息，需登录。
- `POST /chat/messages`：发送客服消息，需登录。
- `GET /ws/chat`：小程序 WebSocket 客服通道。

旧地址、家属、配菜、订单、支付、门店和旧服务详情接口已删除。
