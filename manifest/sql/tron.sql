-- 匹配模式表
CREATE TABLE IF NOT EXISTS `tron_pattern` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `pattern` varchar(255) NOT NULL COMMENT '匹配模式',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_pattern` (`pattern`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tron地址匹配模式表';

-- 匹配结果表
CREATE TABLE IF NOT EXISTS `tron_result` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `address` varchar(255) NOT NULL COMMENT '匹配到的地址',
    `private_key` varchar(255) NOT NULL COMMENT '私钥',
    `pattern_id` bigint(20) NOT NULL COMMENT '关联的模式ID',
    `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_pattern_id` (`pattern_id`),
    KEY `idx_address` (`address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Tron地址匹配结果表';