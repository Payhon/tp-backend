-- FEAT-0010: 组织快捷菜单改为独立路由页面（不再通过机构管理查询参数区分）

-- 1) 组织管理主菜单：修正到真实路由
UPDATE public.sys_ui_elements
SET
    param1 = '/bms/org',
    multilingual = 'route.bms_org',
    route_path = 'view.bms_org'
WHERE id = 'org-management-main';

-- 2) PACK 厂家管理：改为独立路由
UPDATE public.sys_ui_elements
SET
    param1 = '/bms/org/pack-factory',
    multilingual = 'route.bms_org_pack-factory',
    route_path = 'view.bms_org_pack-factory'
WHERE id = 'org-pack-factory';

-- 3) 经销商管理：改为独立路由
UPDATE public.sys_ui_elements
SET
    param1 = '/bms/org/dealer',
    multilingual = 'route.bms_org_dealer',
    route_path = 'view.bms_org_dealer'
WHERE id = 'org-dealer';

-- 4) 门店管理：改为独立路由
UPDATE public.sys_ui_elements
SET
    param1 = '/bms/org/store',
    multilingual = 'route.bms_org_store',
    route_path = 'view.bms_org_store'
WHERE id = 'org-store';
