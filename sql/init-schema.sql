-- 永和护理 MIS 精简版数据库结构
-- 业务边界：内部账户、用户、装修、服务项目、预约、FAQ、阿姨、客服在线。
-- 可重复执行；历史垃圾表请另行执行 sql/cleanup-legacy-tables.sql。

CREATE TABLE IF NOT EXISTS admins (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(50) NOT NULL,
    email VARCHAR(128) NOT NULL,
    role_id BIGINT NULL,
    is_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_login_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_admins_username (username),
    KEY idx_admins_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(32) NOT NULL,
    openid VARCHAR(128) DEFAULT NULL,
    douyin_openid VARCHAR(128) DEFAULT NULL,
    alipay_user_id VARCHAR(128) DEFAULT NULL,
    avatar VARCHAR(512) NOT NULL DEFAULT '',
    nickname VARCHAR(64) NOT NULL,
    phone VARCHAR(32) DEFAULT NULL,
    signature VARCHAR(255) NOT NULL DEFAULT '',
    last_login_at VARCHAR(32) DEFAULT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY idx_users_openid (openid),
    UNIQUE KEY idx_users_douyin_openid (douyin_openid),
    UNIQUE KEY idx_users_alipay_user_id (alipay_user_id),
    UNIQUE KEY idx_users_phone (phone),
    KEY idx_users_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS app_configs (
    `key` VARCHAR(128) NOT NULL,
    value LONGTEXT NOT NULL,
    note VARCHAR(255) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS mini_service_categories (
    id VARCHAR(32) NOT NULL,
    name VARCHAR(64) NOT NULL,
    subtitle VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT NOT NULL,
    icon_url VARCHAR(512) NOT NULL DEFAULT '',
    tags TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_mini_service_categories_enabled_sort (enabled, sort)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS caregivers (
    id VARCHAR(40) NOT NULL,
    application_id VARCHAR(32) DEFAULT NULL,
    contact_phone VARCHAR(32) NOT NULL DEFAULT '',
    avatar_url VARCHAR(512) NOT NULL DEFAULT '',
    name VARCHAR(64) NOT NULL,
    age INT NOT NULL,
    experience_years INT NOT NULL DEFAULT 0,
    origin VARCHAR(128) NOT NULL DEFAULT '',
    service_ids TEXT NOT NULL,
    jobs TEXT NOT NULL,
    availability_status VARCHAR(32) NOT NULL,
    rating DECIMAL(2,1) NOT NULL DEFAULT 0,
    service_count INT NOT NULL DEFAULT 0,
    recommended BOOLEAN NOT NULL DEFAULT FALSE,
    introduction TEXT NOT NULL,
    education VARCHAR(64) NOT NULL DEFAULT '',
    ethnicity VARCHAR(64) NOT NULL DEFAULT '',
    zodiac VARCHAR(32) NOT NULL DEFAULT '',
    birth_date VARCHAR(10) NOT NULL DEFAULT '',
    constellation VARCHAR(32) NOT NULL DEFAULT '',
    skills TEXT NOT NULL,
    certificates TEXT NOT NULL,
    identity_verified BOOLEAN NOT NULL DEFAULT FALSE,
    physical_exam_verified BOOLEAN NOT NULL DEFAULT FALSE,
    medical_report_image_urls TEXT NOT NULL,
    personal_info TEXT NOT NULL,
    work_history TEXT NOT NULL,
    photo_urls TEXT NOT NULL,
    display_fields TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    source VARCHAR(20) NOT NULL DEFAULT 'ADMIN',
    sort INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_caregivers_application_id (application_id),
    KEY idx_caregivers_name (name),
    KEY idx_caregivers_availability (availability_status),
    KEY idx_caregivers_recommended (recommended),
    KEY idx_caregivers_status_sort (status, sort),
    KEY idx_caregivers_source (source)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS demands (
    id VARCHAR(32) NOT NULL,
    user_id VARCHAR(40) DEFAULT NULL,
    service_id VARCHAR(32) NOT NULL,
    service_name VARCHAR(64) NOT NULL,
    caregiver_id VARCHAR(40) DEFAULT NULL,
    caregiver_name VARCHAR(64) NOT NULL DEFAULT '',
    contact_name VARCHAR(64) NOT NULL DEFAULT '',
    contact_phone VARCHAR(32) NOT NULL,
    requirements TEXT NOT NULL,
    source VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING_CONTACT',
    assigned_admin_id BIGINT UNSIGNED DEFAULT NULL,
    idempotency_key VARCHAR(128) DEFAULT NULL,
    submission_scope VARCHAR(128) DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_demands_user_id (user_id),
    KEY idx_demands_service_id (service_id),
    KEY idx_demands_caregiver_id (caregiver_id),
    KEY idx_demands_phone (contact_phone),
    KEY idx_demands_status_created (status, created_at),
    KEY idx_demands_idempotency (submission_scope, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS resumes (
    id VARCHAR(32) NOT NULL,
    user_id VARCHAR(40) DEFAULT NULL,
    intention_service_id VARCHAR(32) NOT NULL,
    service_name VARCHAR(64) NOT NULL,
    work_status VARCHAR(32) NOT NULL,
    experience_range VARCHAR(32) NOT NULL,
    entry_year INT NOT NULL,
    contact_name VARCHAR(64) NOT NULL DEFAULT '',
    contact_phone VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING_CONTACT',
    assigned_admin_id BIGINT UNSIGNED DEFAULT NULL,
    idempotency_key VARCHAR(128) DEFAULT NULL,
    submission_scope VARCHAR(128) DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_resumes_user_id (user_id),
    KEY idx_resumes_service_id (intention_service_id),
    KEY idx_resumes_phone (contact_phone),
    KEY idx_resumes_status_created (status, created_at),
    KEY idx_resumes_idempotency (submission_scope, idempotency_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS business_status_histories (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    entity_type VARCHAR(32) NOT NULL,
    entity_id VARCHAR(32) NOT NULL,
    from_status VARCHAR(32) NOT NULL,
    to_status VARCHAR(32) NOT NULL,
    operator_id BIGINT UNSIGNED DEFAULT NULL,
    note TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_business_history_entity (entity_type, entity_id, created_at),
    KEY idx_business_history_operator (operator_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS faqs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    question VARCHAR(255) NOT NULL,
    answer TEXT NOT NULL,
    category VARCHAR(64) NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    visible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_faqs_visible_sort (visible, sort_order),
    KEY idx_faqs_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS chat_sessions (
    id VARCHAR(40) NOT NULL,
    user_id VARCHAR(40) DEFAULT NULL,
    user_name VARCHAR(64) DEFAULT NULL,
    user_avatar VARCHAR(512) DEFAULT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'open',
    last_message VARCHAR(255) DEFAULT NULL,
    unread_count INT NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_chat_sessions_status_updated (status, updated_at),
    KEY idx_chat_sessions_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    session_id VARCHAR(40) NOT NULL,
    sender VARCHAR(20) NOT NULL,
    msg_type VARCHAR(20) NOT NULL DEFAULT 'text',
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_chat_messages_session_created (session_id, created_at),
    KEY idx_chat_messages_read (is_read)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
