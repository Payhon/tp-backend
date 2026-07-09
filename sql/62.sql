-- FEAT-0065: 版本更新记录后台管理

CREATE TABLE IF NOT EXISTS public.version_update_records (
    id varchar(36) NOT NULL,
    project varchar(32) NOT NULL,
    version_no varchar(64) NOT NULL,
    release_date date NOT NULL,
    update_content text NOT NULL,
    source varchar(32) NOT NULL DEFAULT 'manual',
    source_ref varchar(128) NULL,
    created_at timestamptz(6) NULL DEFAULT NOW(),
    updated_at timestamptz(6) NULL DEFAULT NOW(),
    CONSTRAINT version_update_records_pkey PRIMARY KEY (id),
    CONSTRAINT ck_version_update_records_project CHECK (project IN ('MOBILE', 'CLOUD_FRONTEND', 'CLOUD_BACKEND'))
);

COMMENT ON TABLE public.version_update_records IS '版本更新记录';
COMMENT ON COLUMN public.version_update_records.project IS '项目：MOBILE 移动端；CLOUD_FRONTEND 云平台-WEB前端；CLOUD_BACKEND 云平台-后端';
COMMENT ON COLUMN public.version_update_records.version_no IS '版本号；移动端为 versionName / versionCode，云平台为短 commit hash';
COMMENT ON COLUMN public.version_update_records.release_date IS '发布日期或提交日期';
COMMENT ON COLUMN public.version_update_records.update_content IS '更新内容';
COMMENT ON COLUMN public.version_update_records.source IS '来源：manual/app_update_doc/git_log';
COMMENT ON COLUMN public.version_update_records.source_ref IS '来源引用：文档路径或 commit hash';

CREATE UNIQUE INDEX IF NOT EXISTS uq_version_update_records_project_version_date
    ON public.version_update_records (project, version_no, release_date);

CREATE INDEX IF NOT EXISTS idx_version_update_records_project_date
    ON public.version_update_records (project, release_date DESC);

DO $$
DECLARE
    bms_system_id varchar(36) := 'e1ebd134-53df-3105-35f4-489fc674d173';
BEGIN
    SELECT id INTO bms_system_id
    FROM public.sys_ui_elements
    WHERE element_code = 'bms_system'
    LIMIT 1;

    IF bms_system_id IS NULL THEN
        bms_system_id := 'e1ebd134-53df-3105-35f4-489fc674d173';
    END IF;

    IF EXISTS (
        SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_system_version-updates'
    ) THEN
        UPDATE public.sys_ui_elements
        SET
            parent_id = bms_system_id,
            element_type = 3,
            orders = 48,
            param1 = '/bms/system/version-updates',
            param2 = 'mdi:history',
            param3 = 'self',
            authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            description = '版本更新记录',
            remark = '',
            multilingual = 'route.bms_system_version-updates',
            route_path = 'view.bms_system_version-updates'
        WHERE element_code = 'bms_system_version-updates';
    ELSIF EXISTS (
        SELECT 1 FROM public.sys_ui_elements WHERE element_code = 'bms_system_version_updates'
    ) THEN
        UPDATE public.sys_ui_elements
        SET
            parent_id = bms_system_id,
            element_type = 3,
            orders = 48,
            param1 = '/bms/system/version-updates',
            param2 = 'mdi:history',
            param3 = 'self',
            authority = '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            description = '版本更新记录',
            remark = '',
            multilingual = 'route.bms_system_version-updates',
            route_path = 'view.bms_system_version-updates'
        WHERE element_code = 'bms_system_version_updates';
    ELSE
        INSERT INTO public.sys_ui_elements (
            id, parent_id, element_code, element_type, orders,
            param1, param2, param3, authority, description,
            created_at, remark, multilingual, route_path
        ) VALUES (
            'c82a6d10-7798-4fcb-93d1-0065a0000001',
            bms_system_id,
            'bms_system_version-updates',
            3,
            48,
            '/bms/system/version-updates',
            'mdi:history',
            'self',
            '["TENANT_ADMIN","SYS_ADMIN"]'::json,
            '版本更新记录',
            NOW(),
            '',
            'route.bms_system_version-updates',
            'view.bms_system_version-updates'
        );
    END IF;
END $$;

INSERT INTO public.version_update_records (
    id, project, version_no, release_date, update_content, source, source_ref, created_at, updated_at
) VALUES
  ('vu-mobile-100-20230116', 'MOBILE', '1.0.0 / 100', DATE '2023-01-16', '时间范围：2023-01-16 至 2023-09-08。初始物联网平台移动端；支持设备开关操作、设备在线/离线状态、服务器地址配置、设备当前值展示；加入告警模块；完成场景联动条件、新增/编辑场景、时间条件校验、通知组联动；新增设备监控、折线图、数值显示、日志显示、精度控制，并持续优化 UI、排版、导航高度和文本展示。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-052-20230912', 'MOBILE', '0.5.2 / 100', DATE '2023-09-12', '时间范围：2023-09-12 至 2023-11-29。持续优化界面和文本显示；支持历史数据聚合展示、3 小时历史数据和 tooltip；修复更新数据时操作设备回显异常；补充设备状态展示；修复登录、历史曲线、退出登录后 token 过期提示反复弹出等问题；优化设备详情白色背景、设备列表拉取数量、日志列表指令展示、场景管理激活和开关适配。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-053-20231206', 'MOBILE', '0.5.3 / 101', DATE '2023-12-06', '时间范围：2023-12-06 至 2023-12-07。修复自定义服务地址相关问题；新增告警处理弹窗。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-100-20240913', 'MOBILE', '1.0.0 / 101', DATE '2024-09-13', '时间范围：2024-09-13 至 2026-04-07。切换新 API 和新版设备详情页；优化设备列表、登录失败提示、默认服务器地址、注册和账号注销；建立国际化框架并补齐组件、页面、第三方库和设备详情多语言；加入通知处理和告警详情；完成 Vue2 到 Vue3 迁移，重建富嘉 BMS 登录注册、首页、告警、电池设备详情和我的页面；支持 APP 原生运行、微信小程序蓝牙扫描兼容、扫码添加设备、设备绑定、MQTT 新凭证、Bootloader OTA、MAC 地址区分设备类型、参数权限、组织设备列表、iOS 蓝牙连接、自动升级检测、BLE 数据上报、WebSocket fallback、Relay 透传、设备详情协议控制、参数面板、遗留设备注册、历史记录、扫码路由、账号安全、原生运行配置和共享文案本地化。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-102-20260407', 'MOBILE', '1.0.2 / 102', DATE '2026-04-07', '时间范围：2026-04-07 至 2026-04-10。新增开发者模式、公开内容链接、登录前置守卫和扫码登录保护；扩展账号与设备文案国际化；优化 BLE 扫描和 OTA 状态展示；完善设置页和开发者模式退出、账号资料展示等体验。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-105-20260410', 'MOBILE', '1.0.5 / 105', DATE '2026-04-10', '时间范围：2026-04-10 至 2026-04-15。更新运行元数据；清理仓库系统文件；加固 iOS 蓝牙扫描和请求流程；头像资源优先使用云端 URL；刷新首页资源和启动配置；新增设备高级工厂动作。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-108-20260415', 'MOBILE', '1.0.8 / 108', DATE '2026-04-15', '时间范围：2026-04-15 至 2026-04-20。补齐 iOS 审核所需权限文案和自检脚本；启用仪表临时会话 OTA 升级；兼容旧版状态寄存器间隔；绑定或开通后复用 BLE 会话，减少重复连接等待。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-109-20260420', 'MOBILE', '1.0.9 / 109', DATE '2026-04-20', '时间范围：2026-04-20 至 2026-05-05。新增头像统一展示和裁切流程；扫码列表显示已添加设备并提高轮询效率；修正旧状态寄存器兼容；设备详情新增仪表 OTA 流程和云端遥测兜底；小程序首页图片支持运行态覆盖。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-111-20260505', 'MOBILE', '1.1.1 / 111', DATE '2026-05-05', '时间范围：2026-05-05。加固 BMS 详情实时数据和 OTA 流程；优化协议客户端、帧解析、BLE/MQTT Socket/MQTT WS transport；增强参数页读取、仪表盘展示和 OTA 检测链路稳定性。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-112-20260509', 'MOBILE', '1.1.2 / 112', DATE '2026-05-09', '时间范围：2026-05-09。新增统一接口错误提取；优化 BLE 详情页恢复、绑定错误提示、扫描页提示和添加向导异常处理；提升详情页重连、重试和错误反馈体验。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-113-20260525', 'MOBILE', '1.1.3 / 113', DATE '2026-05-25', '时间范围：2026-05-25 至 2026-06-08。优化电池详情和登录体验；APP 端扫码支持相册二维码识别；简化移动端 4G 连接状态展示；适配 PACK 小程序运行态；添加设备提交连接 MAC 和协议身份 MAC；延长旧遥测在线窗口；放宽 BLE 扫描名称过滤；补齐设备详情参数多语言；支持 4G BMS OTA 透传；放宽 4G BMS OTA 启动超时；支持 4G 实时连接占用降级，第二账号占用时降级为云端只读并提示。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-116-20260624', 'MOBILE', '1.1.6 / 116', DATE '2026-06-24', '时间范围：2026-06-24 至 2026-06-25。优化 UniApp 4G 设备与 OTA 体验；增强 MQTT Socket 休眠唤醒、参数加载和 OTA runtime options；调整首页/登录运行态品牌展示，PACK 未配置专属图时避免默认品牌图闪现；PACK 微信小程序账号注销可跳过当前密码校验。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW()),
  ('vu-mobile-118-20260701', 'MOBILE', '1.1.8 / 118', DATE '2026-07-01', '时间范围：2026-07-01 至 2026-07-07。优化 BMS 详情实时控制；加强 4G MQTT 读取响应匹配、寄存器读取和参数页加载；增加唤醒 probe 与旧响应隔离，避免迟到短响应污染后续读取；新增移动端“我的 > 质保信息”页面、质保 API service、入口图标和中英文文案。', 'app_update_doc', 'docs/APP_UPDATE.md', NOW(), NOW())
ON CONFLICT (project, version_no, release_date) DO UPDATE
SET
    update_content = EXCLUDED.update_content,
    source = EXCLUDED.source,
    source_ref = EXCLUDED.source_ref,
    updated_at = NOW();

INSERT INTO public.version_update_records (
    id, project, version_no, release_date, update_content, source, source_ref, created_at, updated_at
) VALUES
  ('vu-backend-7f1e941d', 'CLOUD_BACKEND', '7f1e941d', DATE '2025-12-01', '添加设备管理相关模型和数据库迁移文件', 'git_log', '7f1e941d', NOW(), NOW()),
  ('vu-backend-3c360e62', 'CLOUD_BACKEND', '3c360e62', DATE '2025-12-02', 'feat(bms): 完成设备转移/绑定/维保 Service 层实现', 'git_log', '3c360e62', NOW(), NOW()),
  ('vu-backend-34ef4262', 'CLOUD_BACKEND', '34ef4262', DATE '2025-12-02', 'feat(bms): 新增设备绑定与维保 API、路由及经销商权限中间件', 'git_log', '34ef4262', NOW(), NOW()),
  ('vu-backend-70f98a3d', 'CLOUD_BACKEND', '70f98a3d', DATE '2025-12-03', 'docs(swagger): regenerate docs for BMS APIs', 'git_log', '70f98a3d', NOW(), NOW()),
  ('vu-backend-ebf61453', 'CLOUD_BACKEND', 'ebf61453', DATE '2025-12-12', 'feat(bms): add BMS menu seed and align dev tooling', 'git_log', 'ebf61453', NOW(), NOW()),
  ('vu-backend-69973b7c', 'CLOUD_BACKEND', '69973b7c', DATE '2025-12-12', 'feat(bms): 支持电池列表查询（含经销商数据隔离）', 'git_log', '69973b7c', NOW(), NOW()),
  ('vu-backend-bad154f5', 'CLOUD_BACKEND', 'bad154f5', DATE '2025-12-13', 'feat(bms): add dashboard, end-user, battery tools and dealer auth template', 'git_log', 'bad154f5', NOW(), NOW()),
  ('vu-backend-bbc1bb6a', 'CLOUD_BACKEND', 'bbc1bb6a', DATE '2025-12-13', 'feat(bms): add activation logs and enrich operation logs', 'git_log', 'bbc1bb6a', NOW(), NOW()),
  ('vu-backend-0761db7f', 'CLOUD_BACKEND', '0761db7f', DATE '2025-12-14', 'feat(bms): add battery maintenance records and API', 'git_log', '0761db7f', NOW(), NOW()),
  ('vu-backend-cfbf8a15', 'CLOUD_BACKEND', 'cfbf8a15', DATE '2025-12-14', 'feat(bms): support system user management with dealer linkage', 'git_log', 'cfbf8a15', NOW(), NOW()),
  ('vu-backend-4a7c725a', 'CLOUD_BACKEND', '4a7c725a', DATE '2025-12-14', 'chore: format users http structs', 'git_log', '4a7c725a', NOW(), NOW()),
  ('vu-backend-a803d94f', 'CLOUD_BACKEND', 'a803d94f', DATE '2025-12-14', 'feat(bms): add battery tag management and assignment', 'git_log', 'a803d94f', NOW(), NOW()),
  ('vu-backend-5ae28db9', 'CLOUD_BACKEND', '5ae28db9', DATE '2025-12-14', 'chore: trim trailing newlines in tag module', 'git_log', '5ae28db9', NOW(), NOW()),
  ('vu-backend-52975841', 'CLOUD_BACKEND', '52975841', DATE '2025-12-14', 'feat(bms): add offline command tasks and auto execution', 'git_log', '52975841', NOW(), NOW()),
  ('vu-backend-9e347861', 'CLOUD_BACKEND', '9e347861', DATE '2025-12-14', 'feat(bms): add battery batch command endpoint', 'git_log', '9e347861', NOW(), NOW()),
  ('vu-backend-13f3285b', 'CLOUD_BACKEND', '13f3285b', DATE '2025-12-14', 'feat(bms): support batch OTA push from battery list', 'git_log', '13f3285b', NOW(), NOW()),
  ('vu-backend-98495b81', 'CLOUD_BACKEND', '98495b81', DATE '2025-12-14', 'chore: format battery http structs', 'git_log', '98495b81', NOW(), NOW()),
  ('vu-backend-dfbdf505', 'CLOUD_BACKEND', 'dfbdf505', DATE '2025-12-14', 'feat(bms): add OTA menus and dealer template access', 'git_log', 'dfbdf505', NOW(), NOW()),
  ('vu-backend-71dc46ef', 'CLOUD_BACKEND', '71dc46ef', DATE '2025-12-14', 'feat(bms): add battery param read/write endpoints', 'git_log', '71dc46ef', NOW(), NOW()),
  ('vu-backend-b81b0e55', 'CLOUD_BACKEND', 'b81b0e55', DATE '2025-12-14', 'feat(bms): expose param set logs under battery routes', 'git_log', 'b81b0e55', NOW(), NOW()),
  ('vu-backend-1dbc25cc', 'CLOUD_BACKEND', '1dbc25cc', DATE '2025-12-25', 'feat: 实现多层级组织架构（Org Tree）改造', 'git_log', '1dbc25cc', NOW(), NOW()),
  ('vu-backend-bd587fdc', 'CLOUD_BACKEND', 'bd587fdc', DATE '2025-12-26', 'feat: 实现APP认证体系和组织管理功能增强', 'git_log', 'bd587fdc', NOW(), NOW()),
  ('vu-backend-b5c50b53', 'CLOUD_BACKEND', 'b5c50b53', DATE '2025-12-27', 'refactor: 优化BMS Dashboard和终端用户查询逻辑', 'git_log', 'b5c50b53', NOW(), NOW()),
  ('vu-backend-a994189b', 'CLOUD_BACKEND', 'a994189b', DATE '2025-12-27', 'feat: 新增APP管理和内容管理功能', 'git_log', 'a994189b', NOW(), NOW()),
  ('vu-backend-90986da8', 'CLOUD_BACKEND', '90986da8', DATE '2025-12-27', 'chore: 更新默认系统管理员账号信息', 'git_log', '90986da8', NOW(), NOW()),
  ('vu-backend-f44e9d6d', 'CLOUD_BACKEND', 'f44e9d6d', DATE '2025-12-28', 'feat: 新增文件存储抽象和管理功能', 'git_log', 'f44e9d6d', NOW(), NOW()),
  ('vu-backend-209c93d6', 'CLOUD_BACKEND', '209c93d6', DATE '2025-12-30', 'docs: 更新设备接入文档和新增BMS设备接入指南', 'git_log', '209c93d6', NOW(), NOW()),
  ('vu-backend-98bccace', 'CLOUD_BACKEND', '98bccace', DATE '2026-01-09', 'feat: 新增APP电池设备管理和MQTT透传功能', 'git_log', '98bccace', NOW(), NOW()),
  ('vu-backend-ff492ffc', 'CLOUD_BACKEND', 'ff492ffc', DATE '2026-01-10', 'feat: 新增移动端设备配置(扫码/蓝牙绑定)功能', 'git_log', 'ff492ffc', NOW(), NOW()),
  ('vu-backend-cdd67f36', 'CLOUD_BACKEND', 'cdd67f36', DATE '2026-01-10', 'feat: 电池管理功能增强和运营日志', 'git_log', 'cdd67f36', NOW(), NOW()),
  ('vu-backend-097c03a3', 'CLOUD_BACKEND', '097c03a3', DATE '2026-01-12', 'feat(battery): 添加产品规格和订单编号字段', 'git_log', '097c03a3', NOW(), NOW()),
  ('vu-backend-028ec195', 'CLOUD_BACKEND', '028ec195', DATE '2026-01-12', 'feat: 添加BMS MQTT透传桥接服务和设备模拟器', 'git_log', '028ec195', NOW(), NOW()),
  ('vu-backend-e1edd2a7', 'CLOUD_BACKEND', 'e1edd2a7', DATE '2026-01-12', 'feat: 新增 bms 通讯类型字段', 'git_log', 'e1edd2a7', NOW(), NOW()),
  ('vu-backend-688d430f', 'CLOUD_BACKEND', '688d430f', DATE '2026-01-14', 'chore: 代码格式化 (go fmt)', 'git_log', '688d430f', NOW(), NOW()),
  ('vu-backend-6ba6f72e', 'CLOUD_BACKEND', '6ba6f72e', DATE '2026-01-14', 'feat: 字典系统支持租户和类别管理', 'git_log', '6ba6f72e', NOW(), NOW()),
  ('vu-backend-9bdadd07', 'CLOUD_BACKEND', '9bdadd07', DATE '2026-01-14', 'feat: 添加字典管理菜单项', 'git_log', '9bdadd07', NOW(), NOW()),
  ('vu-backend-6a6e9e61', 'CLOUD_BACKEND', '6a6e9e61', DATE '2026-01-14', 'feat: 完善设备添加接口', 'git_log', '6a6e9e61', NOW(), NOW()),
  ('vu-backend-6c19a027', 'CLOUD_BACKEND', '6c19a027', DATE '2026-01-15', 'feat: 添加 BMS 通信类型和蓝牙 MAC 地址字段', 'git_log', '6c19a027', NOW(), NOW()),
  ('vu-backend-8181df57', 'CLOUD_BACKEND', '8181df57', DATE '2026-01-16', 'feat: 新增MQTT HTTP认证和APP端MQTT凭证接口', 'git_log', '8181df57', NOW(), NOW()),
  ('vu-backend-bdb3e9e2', 'CLOUD_BACKEND', 'bdb3e9e2', DATE '2026-01-16', 'feat: 新增APP端OTA升级检查接口', 'git_log', 'bdb3e9e2', NOW(), NOW()),
  ('vu-backend-00d2140a', 'CLOUD_BACKEND', '00d2140a', DATE '2026-01-17', 'refactor(service): replace map-based DB inserts with struct-based inserts', 'git_log', '00d2140a', NOW(), NOW()),
  ('vu-backend-942e3fd5', 'CLOUD_BACKEND', '942e3fd5', DATE '2026-01-20', 'feat: 移除 bin文件版本控制', 'git_log', '942e3fd5', NOW(), NOW()),
  ('vu-backend-0d9f2d0a', 'CLOUD_BACKEND', '0d9f2d0a', DATE '2026-01-22', 'feat: 电池生命周期操作接口', 'git_log', '0d9f2d0a', NOW(), NOW()),
  ('vu-backend-4b7bca96', 'CLOUD_BACKEND', '4b7bca96', DATE '2026-01-24', 'feat: add org type device param permission api', 'git_log', '4b7bca96', NOW(), NOW()),
  ('vu-backend-2172b109', 'CLOUD_BACKEND', '2172b109', DATE '2026-01-24', 'feat: add app org device list and options', 'git_log', '2172b109', NOW(), NOW()),
  ('vu-backend-9889f6bc', 'CLOUD_BACKEND', '9889f6bc', DATE '2026-01-24', 'feat: add org type filters for battery list', 'git_log', '9889f6bc', NOW(), NOW()),
  ('vu-backend-43c140ca', 'CLOUD_BACKEND', '43c140ca', DATE '2026-01-24', 'fix: remove middleware import cycle', 'git_log', '43c140ca', NOW(), NOW()),
  ('vu-backend-43585494', 'CLOUD_BACKEND', '43585494', DATE '2026-01-24', 'fix: allow org users to access app battery detail', 'git_log', '43585494', NOW(), NOW()),
  ('vu-backend-f4e50dd2', 'CLOUD_BACKEND', 'f4e50dd2', DATE '2026-01-28', 'feat: 完善 BMS 设备认证信息获取接口', 'git_log', 'f4e50dd2', NOW(), NOW()),
  ('vu-backend-b67dfa63', 'CLOUD_BACKEND', 'b67dfa63', DATE '2026-01-28', 'feat: MQTT 认证增加用户密码认证', 'git_log', 'b67dfa63', NOW(), NOW()),
  ('vu-backend-914fbf63', 'CLOUD_BACKEND', '914fbf63', DATE '2026-02-08', 'Fix write response prefix for func 0', 'git_log', '914fbf63', NOW(), NOW()),
  ('vu-backend-6fe87b35', 'CLOUD_BACKEND', '6fe87b35', DATE '2026-02-12', 'feat: 新增 APP 升级 App 端 API 接口', 'git_log', '6fe87b35', NOW(), NOW()),
  ('vu-backend-9239a857', 'CLOUD_BACKEND', '9239a857', DATE '2026-02-14', 'feat:   add openapi interface', 'git_log', '9239a857', NOW(), NOW()),
  ('vu-backend-8e7ca9d4', 'CLOUD_BACKEND', '8e7ca9d4', DATE '2026-03-02', 'feat: add app battery telemetry report pipeline', 'git_log', '8e7ca9d4', NOW(), NOW()),
  ('vu-backend-9e191810', 'CLOUD_BACKEND', '9e191810', DATE '2026-03-09', 'feat(app): sync ble relay online status', 'git_log', '9e191810', NOW(), NOW()),
  ('vu-backend-23362546', 'CLOUD_BACKEND', '23362546', DATE '2026-03-09', 'fix(dashboard): decouple activated count from online filter', 'git_log', '23362546', NOW(), NOW()),
  ('vu-backend-c8436dcf', 'CLOUD_BACKEND', 'c8436dcf', DATE '2026-03-11', 'feat: expand bms backend relay and admin support', 'git_log', 'c8436dcf', NOW(), NOW()),
  ('vu-backend-57d9b366', 'CLOUD_BACKEND', '57d9b366', DATE '2026-03-11', 'feat: optimize org type permissions and notification config', 'git_log', '57d9b366', NOW(), NOW()),
  ('vu-backend-4b0ecd06', 'CLOUD_BACKEND', '4b0ecd06', DATE '2026-03-17', 'feat: add battery info completion and backoffice role refactor', 'git_log', '4b0ecd06', NOW(), NOW()),
  ('vu-backend-6df184e0', 'CLOUD_BACKEND', '6df184e0', DATE '2026-03-17', 'feat: auto-register legacy provisioned bms devices', 'git_log', '6df184e0', NOW(), NOW()),
  ('vu-backend-ae737f6f', 'CLOUD_BACKEND', 'ae737f6f', DATE '2026-03-17', 'chore: format device provision service', 'git_log', 'ae737f6f', NOW(), NOW()),
  ('vu-backend-0b8d37ff', 'CLOUD_BACKEND', '0b8d37ff', DATE '2026-03-20', 'feat: 电池列表搜索条件改造,增加下拉查询', 'git_log', '0b8d37ff', NOW(), NOW()),
  ('vu-backend-ffdd6a02', 'CLOUD_BACKEND', 'ffdd6a02', DATE '2026-03-21', 'fix: align bms status register parsing', 'git_log', 'ffdd6a02', NOW(), NOW()),
  ('vu-backend-addb4e7f', 'CLOUD_BACKEND', 'addb4e7f', DATE '2026-03-23', 'fix: normalize provision item uuid casing', 'git_log', 'addb4e7f', NOW(), NOW()),
  ('vu-backend-a9d4cd6a', 'CLOUD_BACKEND', 'a9d4cd6a', DATE '2026-03-29', 'feat(app-auth): support account deletion and contact service content', 'git_log', 'a9d4cd6a', NOW(), NOW()),
  ('vu-backend-6e492d5f', 'CLOUD_BACKEND', '6e492d5f', DATE '2026-03-29', 'feat(permission): add mobile ui permission scope', 'git_log', '6e492d5f', NOW(), NOW()),
  ('vu-backend-08ac8faa', 'CLOUD_BACKEND', '08ac8faa', DATE '2026-03-29', 'fix(mqtt): ignore unsupported http auth clients', 'git_log', '08ac8faa', NOW(), NOW()),
  ('vu-backend-bdadb182', 'CLOUD_BACKEND', 'bdadb182', DATE '2026-03-29', 'fix(bms): correct crc scope and bridge payload parsing', 'git_log', 'bdadb182', NOW(), NOW()),
  ('vu-backend-f792da80', 'CLOUD_BACKEND', 'f792da80', DATE '2026-03-29', 'fix(battery): enforce org transfer scope rules', 'git_log', 'f792da80', NOW(), NOW()),
  ('vu-backend-4c2a7a44', 'CLOUD_BACKEND', '4c2a7a44', DATE '2026-04-07', 'feat(app-admin): add public app info and sms diagnostics', 'git_log', '4c2a7a44', NOW(), NOW()),
  ('vu-backend-b29a3068', 'CLOUD_BACKEND', 'b29a3068', DATE '2026-04-10', 'feat(user): add username field and defaults', 'git_log', 'b29a3068', NOW(), NOW()),
  ('vu-backend-54880810', 'CLOUD_BACKEND', '54880810', DATE '2026-04-10', 'feat(upload): validate packages and sign remote ota', 'git_log', '54880810', NOW(), NOW()),
  ('vu-backend-9d48de01', 'CLOUD_BACKEND', '9d48de01', DATE '2026-04-14', 'fix(auth): prefer username for numeric password login', 'git_log', '9d48de01', NOW(), NOW()),
  ('vu-backend-13bf16c6', 'CLOUD_BACKEND', '13bf16c6', DATE '2026-04-14', 'fix(profile): normalize cloud avatar urls', 'git_log', '13bf16c6', NOW(), NOW()),
  ('vu-backend-179297a3', 'CLOUD_BACKEND', '179297a3', DATE '2026-04-15', 'fix(permission): normalize device param permission keys', 'git_log', '179297a3', NOW(), NOW()),
  ('vu-backend-259445a5', 'CLOUD_BACKEND', '259445a5', DATE '2026-04-15', 'feat(ota): allow model-based app battery checks', 'git_log', '259445a5', NOW(), NOW()),
  ('vu-backend-b5839ca3', 'CLOUD_BACKEND', 'b5839ca3', DATE '2026-04-20', 'feat(battery): support auto and batch factory out', 'git_log', 'b5839ca3', NOW(), NOW()),
  ('vu-backend-c45ccae4', 'CLOUD_BACKEND', 'c45ccae4', DATE '2026-04-21', 'feat(battery): add rollback and tighten transfer targets', 'git_log', 'c45ccae4', NOW(), NOW()),
  ('vu-backend-3aa342ad', 'CLOUD_BACKEND', '3aa342ad', DATE '2026-04-27', 'fix(battery): align rollback with parent transfer lineage', 'git_log', '3aa342ad', NOW(), NOW()),
  ('vu-backend-5d49569c', 'CLOUD_BACKEND', '5d49569c', DATE '2026-04-27', 'feat(battery): add detail operation log filtering', 'git_log', '5d49569c', NOW(), NOW()),
  ('vu-backend-146acd9a', 'CLOUD_BACKEND', '146acd9a', DATE '2026-04-27', 'feat(app-battery): add meter ota packages and current telemetry', 'git_log', '146acd9a', NOW(), NOW()),
  ('vu-backend-ff2fcbac', 'CLOUD_BACKEND', 'ff2fcbac', DATE '2026-04-27', 'feat(bms-4g): add bridge comm debug and cloud detail support', 'git_log', 'ff2fcbac', NOW(), NOW()),
  ('vu-backend-bb3a46c4', 'CLOUD_BACKEND', 'bb3a46c4', DATE '2026-05-05', 'fix: align bms status bits and socket sync', 'git_log', 'bb3a46c4', NOW(), NOW()),
  ('vu-backend-e075bf09', 'CLOUD_BACKEND', 'e075bf09', DATE '2026-05-05', 'feat: add battery factory restore action', 'git_log', 'e075bf09', NOW(), NOW()),
  ('vu-backend-af0213d2', 'CLOUD_BACKEND', 'af0213d2', DATE '2026-05-09', 'feat: support 4g module ota package management', 'git_log', 'af0213d2', NOW(), NOW()),
  ('vu-backend-ef2d19f7', 'CLOUD_BACKEND', 'ef2d19f7', DATE '2026-05-25', 'feat: add ota constraints and attachment file APIs', 'git_log', 'ef2d19f7', NOW(), NOW()),
  ('vu-backend-07709609', 'CLOUD_BACKEND', '07709609', DATE '2026-06-01', 'feat: 支持PACK厂小程序配置', 'git_log', '07709609', NOW(), NOW()),
  ('vu-backend-4b1b9a5b', 'CLOUD_BACKEND', '4b1b9a5b', DATE '2026-06-01', 'feat: 支持设备双MAC绑定', 'git_log', '4b1b9a5b', NOW(), NOW()),
  ('vu-backend-1111ed12', 'CLOUD_BACKEND', '1111ed12', DATE '2026-06-01', 'fix: 调整默认在线保活时长', 'git_log', '1111ed12', NOW(), NOW()),
  ('vu-backend-6ec0d8b3', 'CLOUD_BACKEND', '6ec0d8b3', DATE '2026-06-04', 'fix: 过滤BMS历史宽表结构字段', 'git_log', '6ec0d8b3', NOW(), NOW()),
  ('vu-backend-c8ae9f8c', 'CLOUD_BACKEND', 'c8ae9f8c', DATE '2026-06-08', 'fix: 优化双MAC绑定和解绑释放', 'git_log', 'c8ae9f8c', NOW(), NOW()),
  ('vu-backend-65d6bd40', 'CLOUD_BACKEND', '65d6bd40', DATE '2026-06-08', 'fix: 重新解析重建设备的桥接ID', 'git_log', '65d6bd40', NOW(), NOW()),
  ('vu-backend-e85bceb9', 'CLOUD_BACKEND', 'e85bceb9', DATE '2026-06-08', 'feat: 增加APP MQTT透传连接互斥', 'git_log', 'e85bceb9', NOW(), NOW()),
  ('vu-backend-8a0ec362', 'CLOUD_BACKEND', '8a0ec362', DATE '2026-06-24', 'feat: 返回APP首页设备IMEI标识', 'git_log', '8a0ec362', NOW(), NOW()),
  ('vu-backend-4a64ae71', 'CLOUD_BACKEND', '4a64ae71', DATE '2026-06-24', 'feat: 增加4G BMS直连OTA检测接口', 'git_log', '4a64ae71', NOW(), NOW()),
  ('vu-backend-9e1d7b53', 'CLOUD_BACKEND', '9e1d7b53', DATE '2026-06-24', 'feat: 增强4G BMS BOOT链路诊断', 'git_log', '9e1d7b53', NOW(), NOW()),
  ('vu-backend-4cbc92e0', 'CLOUD_BACKEND', '4cbc92e0', DATE '2026-06-25', 'feat: allow pack wxmp account deletion without password', 'git_log', '4cbc92e0', NOW(), NOW()),
  ('vu-backend-bad5b7c8', 'CLOUD_BACKEND', 'bad5b7c8', DATE '2026-07-01', 'feat: add charge discharge factory permissions', 'git_log', 'bad5b7c8', NOW(), NOW()),
  ('vu-backend-201640bd', 'CLOUD_BACKEND', '201640bd', DATE '2026-07-07', 'fix: add bms mqtt socket frame diagnostics', 'git_log', '201640bd', NOW(), NOW()),
  ('vu-backend-697e546c', 'CLOUD_BACKEND', '697e546c', DATE '2026-07-07', 'feat: add warranty info and pack wxmp config', 'git_log', '697e546c', NOW(), NOW()),
  ('vu-backend-885b9b13', 'CLOUD_BACKEND', '885b9b13', DATE '2026-07-07', 'fix: mark 4g battery online from socket replies', 'git_log', '885b9b13', NOW(), NOW())
ON CONFLICT (project, version_no, release_date) DO UPDATE
SET
    update_content = EXCLUDED.update_content,
    source = EXCLUDED.source,
    source_ref = EXCLUDED.source_ref,
    updated_at = NOW();

INSERT INTO public.version_update_records (
    id, project, version_no, release_date, update_content, source, source_ref, created_at, updated_at
) VALUES
  ('vu-frontend-a5861baf', 'CLOUD_FRONTEND', 'a5861baf', DATE '2025-12-03', 'feat(bms-web): 新增BMS管理菜单及维保中心页面', 'git_log', 'a5861baf', NOW(), NOW()),
  ('vu-frontend-e424dbfb', 'CLOUD_FRONTEND', 'e424dbfb', DATE '2025-12-12', 'feat: 初步BMS 开发完成', 'git_log', 'e424dbfb', NOW(), NOW()),
  ('vu-frontend-40e73d66', 'CLOUD_FRONTEND', '40e73d66', DATE '2025-12-12', 'feat(bms-web): 增加电池列表页面与接口对接', 'git_log', '40e73d66', NOW(), NOW()),
  ('vu-frontend-7474652b', 'CLOUD_FRONTEND', '7474652b', DATE '2025-12-13', 'feat(bms-web): add dashboard/end-user pages and dealer auth template', 'git_log', '7474652b', NOW(), NOW()),
  ('vu-frontend-c907cbd4', 'CLOUD_FRONTEND', 'c907cbd4', DATE '2025-12-13', 'feat(bms-web): add activation log and operation log pages', 'git_log', 'c907cbd4', NOW(), NOW()),
  ('vu-frontend-95e8b53e', 'CLOUD_FRONTEND', '95e8b53e', DATE '2025-12-14', 'feat(bms-web): add battery maintenance tab in warranty center', 'git_log', '95e8b53e', NOW(), NOW()),
  ('vu-frontend-99c1f930', 'CLOUD_FRONTEND', '99c1f930', DATE '2025-12-14', 'feat(bms-web): add BMS system user/role pages', 'git_log', '99c1f930', NOW(), NOW()),
  ('vu-frontend-24a6d045', 'CLOUD_FRONTEND', '24a6d045', DATE '2025-12-14', 'feat(bms-web): add battery tag management and batch tagging', 'git_log', '24a6d045', NOW(), NOW()),
  ('vu-frontend-2b496a27', 'CLOUD_FRONTEND', '2b496a27', DATE '2025-12-14', 'feat(bms-web): add offline command page and store action', 'git_log', '2b496a27', NOW(), NOW()),
  ('vu-frontend-8d6100db', 'CLOUD_FRONTEND', '8d6100db', DATE '2025-12-14', 'feat(bms-web): add batch command modal in battery list', 'git_log', '8d6100db', NOW(), NOW()),
  ('vu-frontend-9afded0f', 'CLOUD_FRONTEND', '9afded0f', DATE '2025-12-14', 'feat(bms-web): add batch OTA push modal and task detail view', 'git_log', '9afded0f', NOW(), NOW()),
  ('vu-frontend-17c82515', 'CLOUD_FRONTEND', '17c82515', DATE '2025-12-14', 'feat(bms-web): add OTA package and task management pages', 'git_log', '17c82515', NOW(), NOW()),
  ('vu-frontend-c3549d16', 'CLOUD_FRONTEND', 'c3549d16', DATE '2025-12-14', 'feat(bms-web): upload firmware when creating OTA package', 'git_log', 'c3549d16', NOW(), NOW()),
  ('vu-frontend-9a5337c8', 'CLOUD_FRONTEND', '9a5337c8', DATE '2025-12-14', 'feat(bms-web): improve OTA package upload and add param modal', 'git_log', '9a5337c8', NOW(), NOW()),
  ('vu-frontend-3f394d09', 'CLOUD_FRONTEND', '3f394d09', DATE '2025-12-14', 'feat(bms-web): show param set logs in battery param modal', 'git_log', '3f394d09', NOW(), NOW()),
  ('vu-frontend-b11b38e5', 'CLOUD_FRONTEND', 'b11b38e5', DATE '2025-12-25', 'fix: 修复 BMS 管理模块路由和编译错误', 'git_log', 'b11b38e5', NOW(), NOW()),
  ('vu-frontend-4fcc3ac7', 'CLOUD_FRONTEND', '4fcc3ac7', DATE '2025-12-26', 'feat: 优化组织管理UI和短信配置功能', 'git_log', '4fcc3ac7', NOW(), NOW()),
  ('vu-frontend-88387d2f', 'CLOUD_FRONTEND', '88387d2f', DATE '2025-12-27', 'feat: 新增APP管理前端页面和优化权限管理', 'git_log', '88387d2f', NOW(), NOW()),
  ('vu-frontend-e755f69a', 'CLOUD_FRONTEND', 'e755f69a', DATE '2025-12-28', 'feat: 新增文件选择器和存储设置功能', 'git_log', 'e755f69a', NOW(), NOW()),
  ('vu-frontend-36285b62', 'CLOUD_FRONTEND', '36285b62', DATE '2025-12-30', 'style: 优化文件选择器组件UI', 'git_log', '36285b62', NOW(), NOW()),
  ('vu-frontend-b426dc21', 'CLOUD_FRONTEND', 'b426dc21', DATE '2026-01-09', 'style: 代码格式优化（空行调整）', 'git_log', 'b426dc21', NOW(), NOW()),
  ('vu-frontend-fdc2c0a0', 'CLOUD_FRONTEND', 'fdc2c0a0', DATE '2026-01-10', 'feat: 电池管理功能增强和UI优化', 'git_log', 'fdc2c0a0', NOW(), NOW()),
  ('vu-frontend-9f4a631a', 'CLOUD_FRONTEND', '9f4a631a', DATE '2026-01-12', 'feat(battery): 添加产品规格和订单编号字段', 'git_log', '9f4a631a', NOW(), NOW()),
  ('vu-frontend-a2b146d3', 'CLOUD_FRONTEND', 'a2b146d3', DATE '2026-01-14', 'feat: 添加BMS协议库和面板组件', 'git_log', 'a2b146d3', NOW(), NOW()),
  ('vu-frontend-20a542e3', 'CLOUD_FRONTEND', '20a542e3', DATE '2026-01-14', 'feat: 新增字典管理功能和工具函数', 'git_log', '20a542e3', NOW(), NOW()),
  ('vu-frontend-ddca4288', 'CLOUD_FRONTEND', 'ddca4288', DATE '2026-01-14', 'feat: 增强字典管理功能和新增语言选择组件', 'git_log', 'ddca4288', NOW(), NOW()),
  ('vu-frontend-a09439e7', 'CLOUD_FRONTEND', 'a09439e7', DATE '2026-01-17', 'feat(content): add image upload component and markdown editor fullscreen', 'git_log', 'a09439e7', NOW(), NOW()),
  ('vu-frontend-d05e7d2d', 'CLOUD_FRONTEND', 'd05e7d2d', DATE '2026-01-24', 'feat: add lifecycle actions and param permissions', 'git_log', 'd05e7d2d', NOW(), NOW()),
  ('vu-frontend-a23ffa04', 'CLOUD_FRONTEND', 'a23ffa04', DATE '2026-01-24', 'feat: scope battery list by org type', 'git_log', 'a23ffa04', NOW(), NOW()),
  ('vu-frontend-f88bc319', 'CLOUD_FRONTEND', 'f88bc319', DATE '2026-01-24', 'fix: use scoped org options for factory out', 'git_log', 'f88bc319', NOW(), NOW()),
  ('vu-frontend-d9e6b81c', 'CLOUD_FRONTEND', 'd9e6b81c', DATE '2026-01-24', 'chore: rename cell count label', 'git_log', 'd9e6b81c', NOW(), NOW()),
  ('vu-frontend-54c6af8b', 'CLOUD_FRONTEND', '54c6af8b', DATE '2026-02-14', 'feat: Add ç¬ openapi management', 'git_log', '54c6af8b', NOW(), NOW()),
  ('vu-frontend-45a6a534', 'CLOUD_FRONTEND', '45a6a534', DATE '2026-03-02', 'feat: show cloud-first bms realtime and history', 'git_log', '45a6a534', NOW(), NOW()),
  ('vu-frontend-c62d8cdd', 'CLOUD_FRONTEND', 'c62d8cdd', DATE '2026-03-03', 'fix: allow cloud-only bms panel for non-4g devices', 'git_log', 'c62d8cdd', NOW(), NOW()),
  ('vu-frontend-06e7ed09', 'CLOUD_FRONTEND', '06e7ed09', DATE '2026-03-03', 'fix: improve cloud connection status and disconnect action', 'git_log', '06e7ed09', NOW(), NOW()),
  ('vu-frontend-b1089d7c', 'CLOUD_FRONTEND', 'b1089d7c', DATE '2026-03-04', 'feat: align bms params with uniapp advanced settings', 'git_log', 'b1089d7c', NOW(), NOW()),
  ('vu-frontend-64155359', 'CLOUD_FRONTEND', '64155359', DATE '2026-03-11', 'feat: extend bms management and protocol settings', 'git_log', '64155359', NOW(), NOW()),
  ('vu-frontend-719f0de2', 'CLOUD_FRONTEND', '719f0de2', DATE '2026-03-11', 'feat: refine bms detail and admin settings', 'git_log', '719f0de2', NOW(), NOW()),
  ('vu-frontend-ce28f7c0', 'CLOUD_FRONTEND', 'ce28f7c0', DATE '2026-03-17', 'feat: update bms battery and backoffice management views', 'git_log', 'ce28f7c0', NOW(), NOW()),
  ('vu-frontend-4611a175', 'CLOUD_FRONTEND', '4611a175', DATE '2026-03-20', 'feat: 电池列表搜索条件改造', 'git_log', '4611a175', NOW(), NOW()),
  ('vu-frontend-f47b2edb', 'CLOUD_FRONTEND', 'f47b2edb', DATE '2026-03-21', 'fix: align bms panel status display', 'git_log', 'f47b2edb', NOW(), NOW()),
  ('vu-frontend-43cdd56d', 'CLOUD_FRONTEND', '43cdd56d', DATE '2026-03-29', 'feat(permission): split mobile ui permissions in admin', 'git_log', '43cdd56d', NOW(), NOW()),
  ('vu-frontend-29b134bb', 'CLOUD_FRONTEND', '29b134bb', DATE '2026-03-29', 'feat(app-manage): add contact service content config', 'git_log', '29b134bb', NOW(), NOW()),
  ('vu-frontend-641d4da6', 'CLOUD_FRONTEND', '641d4da6', DATE '2026-03-29', 'feat(bms-ui): refine transfer targets and cell panel', 'git_log', '641d4da6', NOW(), NOW()),
  ('vu-frontend-818e5bb5', 'CLOUD_FRONTEND', '818e5bb5', DATE '2026-04-07', 'feat(admin): add public app pages and config diagnostics', 'git_log', '818e5bb5', NOW(), NOW()),
  ('vu-frontend-0b84153e', 'CLOUD_FRONTEND', '0b84153e', DATE '2026-04-10', 'feat(upload): add cloud direct upload flow', 'git_log', '0b84153e', NOW(), NOW()),
  ('vu-frontend-70f14c69', 'CLOUD_FRONTEND', '70f14c69', DATE '2026-04-10', 'feat(user): expose username in admin lists', 'git_log', '70f14c69', NOW(), NOW()),
  ('vu-frontend-2092735c', 'CLOUD_FRONTEND', '2092735c', DATE '2026-04-20', 'feat(bms-ui): add batch factory out actions', 'git_log', '2092735c', NOW(), NOW()),
  ('vu-frontend-ceba6f65', 'CLOUD_FRONTEND', 'ceba6f65', DATE '2026-04-20', 'feat(branding): refresh public app branding', 'git_log', 'ceba6f65', NOW(), NOW()),
  ('vu-frontend-f107098c', 'CLOUD_FRONTEND', 'f107098c', DATE '2026-04-21', 'feat(bms-ui): add rollback flow and tighten transfer targets', 'git_log', 'f107098c', NOW(), NOW()),
  ('vu-frontend-dd63d5e5', 'CLOUD_FRONTEND', 'dd63d5e5', DATE '2026-04-27', 'feat(ota-ui): support meter firmware packages', 'git_log', 'dd63d5e5', NOW(), NOW()),
  ('vu-frontend-b073601a', 'CLOUD_FRONTEND', 'b073601a', DATE '2026-04-27', 'feat(bms-ops): add comm debug and detail operation logs', 'git_log', 'b073601a', NOW(), NOW()),
  ('vu-frontend-5b6cb731', 'CLOUD_FRONTEND', '5b6cb731', DATE '2026-04-27', 'feat(bms-panel): support cloud cell telemetry fallback', 'git_log', '5b6cb731', NOW(), NOW()),
  ('vu-frontend-e2ffd3b5', 'CLOUD_FRONTEND', 'e2ffd3b5', DATE '2026-05-05', 'feat: add battery factory restore UI', 'git_log', 'e2ffd3b5', NOW(), NOW()),
  ('vu-frontend-8c91f35a', 'CLOUD_FRONTEND', '8c91f35a', DATE '2026-05-05', 'fix: improve bms realtime protocol panel', 'git_log', '8c91f35a', NOW(), NOW()),
  ('vu-frontend-44e650e4', 'CLOUD_FRONTEND', '44e650e4', DATE '2026-05-09', 'feat: add 4g module tab in ota package page', 'git_log', '44e650e4', NOW(), NOW()),
  ('vu-frontend-10dd9326', 'CLOUD_FRONTEND', '10dd9326', DATE '2026-05-25', 'feat: add attachment management and ota constraint UI', 'git_log', '10dd9326', NOW(), NOW()),
  ('vu-frontend-4bf8b7f7', 'CLOUD_FRONTEND', '4bf8b7f7', DATE '2026-06-01', 'feat: 增加PACK小程序后台配置', 'git_log', '4bf8b7f7', NOW(), NOW()),
  ('vu-frontend-a9deddf3', 'CLOUD_FRONTEND', 'a9deddf3', DATE '2026-06-01', 'feat: 展示设备身份MAC', 'git_log', 'a9deddf3', NOW(), NOW()),
  ('vu-frontend-782af215', 'CLOUD_FRONTEND', '782af215', DATE '2026-06-04', 'fix: 补齐Web BMS模块多语言', 'git_log', '782af215', NOW(), NOW()),
  ('vu-frontend-c3595d7f', 'CLOUD_FRONTEND', 'c3595d7f', DATE '2026-06-05', 'fix: 完善Web BMS管理页多语言', 'git_log', 'c3595d7f', NOW(), NOW()),
  ('vu-frontend-ce219a99', 'CLOUD_FRONTEND', 'ce219a99', DATE '2026-06-08', 'fix: 优化OTA包列表布局', 'git_log', 'ce219a99', NOW(), NOW()),
  ('vu-frontend-4e2003df', 'CLOUD_FRONTEND', '4e2003df', DATE '2026-07-01', 'feat: add bms charge discharge factory actions', 'git_log', '4e2003df', NOW(), NOW()),
  ('vu-frontend-c3a9a189', 'CLOUD_FRONTEND', 'c3a9a189', DATE '2026-07-07', 'feat: add warranty tab and pack wxmp config page', 'git_log', 'c3a9a189', NOW(), NOW()),
  ('vu-frontend-efc78ca9', 'CLOUD_FRONTEND', 'efc78ca9', DATE '2026-07-07', 'fix: render bms cell voltage horizontally', 'git_log', 'efc78ca9', NOW(), NOW())
ON CONFLICT (project, version_no, release_date) DO UPDATE
SET
    update_content = EXCLUDED.update_content,
    source = EXCLUDED.source,
    source_ref = EXCLUDED.source_ref,
    updated_at = NOW();
