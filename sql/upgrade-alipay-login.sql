-- 支付宝小程序登录升级：为已有 users 表增加支付宝稳定用户 ID。
ALTER TABLE users
    ADD COLUMN alipay_user_id VARCHAR(128) NULL AFTER douyin_openid,
    ADD UNIQUE KEY idx_users_alipay_user_id (alipay_user_id),
    MODIFY COLUMN phone VARCHAR(32) NULL;
