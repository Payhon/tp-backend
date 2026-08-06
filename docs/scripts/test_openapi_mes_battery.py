#!/usr/bin/env python3
"""
MES OpenAPI battery integration smoke test.

Usage:
  python3 test_openapi_mes_battery.py \
    --base-url http://fjbms.yz6688.cn \
    --app-id app_xxx \
    --app-secret sk_xxx
"""

from __future__ import annotations

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from typing import Any, Dict, Tuple


def _request_json(
    method: str,
    url: str,
    headers: Dict[str, str],
    body: Dict[str, Any] | None = None,
    timeout: int = 20,
) -> Tuple[int, Dict[str, Any]]:
    data = None
    if body is not None:
        data = json.dumps(body).encode("utf-8")

    req = urllib.request.Request(url=url, data=data, method=method)
    for key, value in headers.items():
        req.add_header(key, value)

    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            status = resp.status
            text = resp.read().decode("utf-8", errors="replace")
            try:
                payload = json.loads(text) if text else {}
            except json.JSONDecodeError:
                payload = {"_raw": text}
            return status, payload
    except urllib.error.HTTPError as e:
        text = e.read().decode("utf-8", errors="replace")
        try:
            payload = json.loads(text) if text else {}
        except json.JSONDecodeError:
            payload = {"_raw": text}
        return e.code, payload


def _must(condition: bool, msg: str) -> None:
    if not condition:
        raise AssertionError(msg)


def main() -> int:
    parser = argparse.ArgumentParser(description="Test MES battery OpenAPI endpoints")
    parser.add_argument("--base-url", default=os.getenv("FJBMS_BASE_URL"), required=False)
    parser.add_argument("--app-id", default=os.getenv("FJBMS_APP_ID"), required=False)
    parser.add_argument("--app-secret", default=os.getenv("FJBMS_APP_SECRET"), required=False)
    parser.add_argument("--serial-number", default=None, required=False)
    parser.add_argument(
        "--reassign-serials",
        default=None,
        help="Comma-separated existing battery serial numbers. When set, run only PACK reassignment.",
    )
    parser.add_argument("--target-pack-factory-name", default=None)
    parser.add_argument("--reassign-remark", default="MES OpenAPI smoke test")
    args = parser.parse_args()

    base_url = (args.base_url or "").rstrip("/")
    app_id = (args.app_id or "").strip()
    app_secret = (args.app_secret or "").strip()

    if not base_url or not app_id or not app_secret:
        print("Missing required config: --base-url --app-id --app-secret", file=sys.stderr)
        return 2

    serial_number = args.serial_number
    if not serial_number:
        serial_number = f"MESAPI{int(time.time() * 1000)}"

    headers = {
        "Content-Type": "application/json",
        "x-app-id": app_id,
        "x-secret-key": app_secret,
    }

    if args.reassign_serials:
        serial_numbers = [item.strip() for item in args.reassign_serials.split(",") if item.strip()]
        target_pack_factory_name = (args.target_pack_factory_name or "").strip()
        if not serial_numbers or not target_pack_factory_name:
            print(
                "--reassign-serials requires at least one SN and --target-pack-factory-name",
                file=sys.stderr,
            )
            return 2

        reassign_url = f"{base_url}/api/v1/openapi/mes/battery/reassign-pack-factory"
        reassign_body = {
            "serial_numbers": serial_numbers,
            "target_pack_factory_name": target_pack_factory_name,
            "remark": args.reassign_remark,
        }
        print(f"[1/1] POST {reassign_url}")
        reassign_http, reassign_resp = _request_json(
            "POST",
            reassign_url,
            headers=headers,
            body=reassign_body,
        )
        print(f"HTTP {reassign_http}")
        print(json.dumps(reassign_resp, ensure_ascii=False, indent=2))

        _must(reassign_http in (200, 201), f"Reassign endpoint http failed: {reassign_http}")
        _must(reassign_resp.get("code") == 200, f"Reassign endpoint business failed: {reassign_resp}")
        reassign_data = reassign_resp.get("data") or {}
        _must(
            reassign_data.get("total") == len(dict.fromkeys(serial_numbers)),
            f"Reassign response total mismatch: {reassign_data}",
        )
        results = reassign_data.get("results") or []
        _must(len(results) == reassign_data.get("total"), f"Reassign result count mismatch: {reassign_data}")
        valid_statuses = {"REASSIGNED", "UNCHANGED", "FAILED"}
        _must(
            all(item.get("status") in valid_statuses for item in results),
            f"Reassign response contains invalid status: {results}",
        )

        for item in results:
            if item.get("status") not in {"REASSIGNED", "UNCHANGED"}:
                continue
            serial = str(item.get("serial_number") or "")
            query_url = (
                f"{base_url}/api/v1/openapi/mes/battery/"
                f"{urllib.parse.quote(serial, safe='')}"
            )
            query_http, query_resp = _request_json("GET", query_url, headers=headers)
            _must(query_http in (200, 201), f"Query after reassign http failed: {query_http}")
            _must(query_resp.get("code") == 200, f"Query after reassign failed: {query_resp}")
            query_data = query_resp.get("data") or {}
            _must(
                query_data.get("owner_org_name") == target_pack_factory_name,
                f"Owner PACK mismatch after reassign: {query_data}",
            )

        print("\nPASS: PACK factory reassignment API is working.")
        return 0

    now = datetime.now(timezone.utc)
    create_body = {
        "item_uuid": serial_number,
        "batch_number": now.strftime("BATCH-%Y%m%d"),
        "product_spec": "51.2V100Ah",
        "order_number": now.strftime("MES-PO-%Y%m%d-%H%M%S"),
        "bms_comm_type": 1,
        "pack_factory_name": "示例PACK厂家",
        "ble_mac": "AC:11:22:33:44:55",
        "production_date": now.strftime("%Y-%m-%d"),
    }

    create_url = f"{base_url}/api/v1/openapi/mes/battery"
    query_url = f"{base_url}/api/v1/openapi/mes/battery/{serial_number}"

    print(f"[1/2] POST {create_url}")
    create_http, create_resp = _request_json("POST", create_url, headers=headers, body=create_body)
    print(f"HTTP {create_http}")
    print(json.dumps(create_resp, ensure_ascii=False, indent=2))

    _must(create_http in (200, 201), f"Create endpoint http failed: {create_http}")
    _must(create_resp.get("code") == 200, f"Create endpoint business failed: {create_resp}")

    create_data = create_resp.get("data") or {}
    created_number = create_data.get("device_number")
    created_uuid = create_data.get("item_uuid")
    _must(
        created_number == serial_number or created_uuid == serial_number,
        f"Create response serial mismatch, expected={serial_number}, got={create_data}",
    )

    print(f"[2/2] GET {query_url}")
    query_http, query_resp = _request_json("GET", query_url, headers=headers)
    print(f"HTTP {query_http}")
    print(json.dumps(query_resp, ensure_ascii=False, indent=2))

    _must(query_http in (200, 201), f"Query endpoint http failed: {query_http}")
    _must(query_resp.get("code") == 200, f"Query endpoint business failed: {query_resp}")

    query_data = query_resp.get("data") or {}
    queried_number = query_data.get("device_number")
    queried_uuid = query_data.get("item_uuid")
    _must(
        queried_number == serial_number or queried_uuid == serial_number,
        f"Query response serial mismatch, expected={serial_number}, got={query_data}",
    )

    print("\nPASS: Create + Query APIs are working.")
    print(f"serial_number={serial_number}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except AssertionError as e:
        print(f"\nFAIL: {e}", file=sys.stderr)
        raise SystemExit(1)
    except Exception as e:  # pragma: no cover
        print(f"\nERROR: {e}", file=sys.stderr)
        raise SystemExit(1)
