-- =============================================================================
-- 加工厂 ERP 数据库模型 · 物理 DDL（MySQL 8.0+）
-- 依据：加工厂ERP逻辑数据模型.md + 框架设计文档第7章权限
-- 字符集：utf8mb4 / InnoDB
-- PK：BIGINT 自增（生产可改为雪花，类型保持 BIGINT）
-- =============================================================================

CREATE DATABASE IF NOT EXISTS erp_factory
  DEFAULT CHARACTER SET utf8mb4
  DEFAULT COLLATE utf8mb4_unicode_ci;

USE erp_factory;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;
