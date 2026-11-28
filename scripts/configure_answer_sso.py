import json
import os
import sys
import time
import urllib.error
import urllib.request


def load_env(path=".env"):
    env = dict(os.environ)
    if os.path.exists(path):
        with open(path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#") or "=" not in line:
                    continue
                k, v = line.split("=", 1)
                env.setdefault(k.strip(), v.strip())
    return env


def api_call(base, path, method="GET", body=None, token=None):
    hdrs = {"Accept": "application/json"}
    data = None
    if token:
        hdrs["Authorization"] = token
    if body is not None:
        data = json.dumps(body).encode("utf-8")
        hdrs["Content-Type"] = "application/json"
    req = urllib.request.Request(base + path, data=data, method=method, headers=hdrs)
    try:
        with urllib.request.urlopen(req, timeout=15) as r:
            return r.status, json.loads(r.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        raw = e.read().decode("utf-8", errors="replace")
        try:
            return e.code, json.loads(raw)
        except Exception:
            return e.code, {"raw": raw}


def wait_for_answer(base, timeout=90):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            st, resp = api_call(base, "/answer/api/v1/siteinfo")
            if st == 200 and isinstance(resp, dict) and resp.get("data"):
                return
        except Exception:
            pass
        time.sleep(2)
    raise RuntimeError("Apache Answer did not become ready in time")


def main():
    env = load_env()
    answer_base = env.get("ANSWER_SITE_URL", "http://localhost:9080").rstrip("/")
    frontend_url = env.get("FRONTEND_URL", "http://localhost:8081").rstrip("/")
    admin_email = env["ANSWER_ADMIN_EMAIL"]
    admin_password = env["ANSWER_ADMIN_PASSWORD"]
    client_id = env["OAUTH_CLIENT_ID"]
    client_secret = env["OAUTH_CLIENT_SECRET"]

    print(f"Waiting for Apache Answer at {answer_base}...")
    wait_for_answer(answer_base)

    st, login_resp = api_call(
        answer_base,
        "/answer/api/v1/user/login/email",
        method="POST",
        body={"e_mail": admin_email, "pass": admin_password},
    )
    assert st == 200 and login_resp.get("data", {}).get("access_token"), f"Answer admin login failed: {st} {login_resp}"
    token = login_resp["data"]["access_token"]
    print("Logged in to Apache Answer admin API")

    # List plugins and locate connector-basic
    st, plugins_resp = api_call(answer_base, "/answer/admin/api/plugins", token=token)
    assert st == 200, f"Failed to list plugins: {st} {plugins_resp}"
    plugins = plugins_resp.get("data") or []
    connector = next((p for p in plugins if "basic" in (p.get("slug_name") or "").lower() or "oauth" in (p.get("slug_name") or "").lower()), None)
    assert connector, f"connector-basic plugin not found in: {plugins}"
    slug = connector["slug_name"]
    print(f"Found OAuth2 connector plugin: slug={slug}, enabled={connector.get('enabled')}")

    if not connector.get("enabled"):
        st, en_resp = api_call(
            answer_base,
            "/answer/admin/api/plugin/status",
            method="PUT",
            body={"plugin_slug_name": slug, "enabled": True},
            token=token,
        )
        assert st == 200, f"Failed to enable plugin {slug}: {st} {en_resp}"
        print(f"Enabled plugin {slug}")

    # Inspect config schema and populate OAuth2 SSO fields
    st, cfg_resp = api_call(
        answer_base,
        f"/answer/admin/api/plugin/config?plugin_slug_name={slug}",
        token=token,
    )
    assert st == 200, f"Failed to get plugin config: {st} {cfg_resp}"
    configs = cfg_resp.get("data", {}).get("config_fields") or []
    field_names = [c.get("name") for c in configs]
    print("Plugin config fields:", field_names)

    desired = {
        "name": "Alumni NTUST",
        "client_id": client_id,
        "client_secret": client_secret,
        "authorize_url": f"{frontend_url}/#/oauth/authorize",
        "token_url": "http://backend:8080/api/oauth/token",
        "user_json_url": "http://backend:8080/api/oauth/userinfo",
        "user_id_json_path": "id",
        "user_display_name_json_path": "display_name",
        "user_username_json_path": "username",
        "user_email_json_path": "email",
        "user_avatar_json_path": "avatar",
        "scope": "openid,profile,email",
        "check_email_verified": False,
        "email_verified_json_path": "",
    }
    payload_configs = {}
    for c in configs:
        name = c.get("name")
        if name in desired:
            payload_configs[name] = desired[name]
        elif "value" in c:
            payload_configs[name] = c["value"]

    st, save_resp = api_call(
        answer_base,
        "/answer/admin/api/plugin/config",
        method="PUT",
        body={"plugin_slug_name": slug, "config_fields": payload_configs},
        token=token,
    )
    assert st == 200, f"Failed to save plugin config: {st} {save_resp}"
    print(f"Configured {slug} SSO successfully")

    st, info_resp = api_call(answer_base, "/answer/api/v1/connector/info")
    assert st == 200 and info_resp.get("data"), f"Connector info not active: {st} {info_resp}"
    print("Active connectors:", info_resp["data"])

    # Seed welcome question if not already present
    st, q_resp = api_call(answer_base, "/answer/api/v1/question/page?page=1&page_size=20&order=newest")
    assert st == 200, f"Failed to list questions: {st} {q_resp}"
    q_list = q_resp.get("data", {}).get("list") or []
    welcome_title = "Selamat Datang di Forum Q&A Alumni NTUST Indonesia"
    if not any(q.get("title") == welcome_title for q in q_list):
        st, qc_resp = api_call(
            answer_base,
            "/answer/api/v1/question",
            method="POST",
            body={
                "title": welcome_title,
                "content": "Forum diskusi kini telah ditingkatkan menggunakan **Apache Answer**.\n\nSilakan login dengan akun Alumni NTUST terverifikasi melalui tombol **Alumni NTUST** untuk bertanya, menjawab, voting, dan berdiskusi berdasarkan topik tag!",
                "tags": [{"slug_name": "pengumuman", "display_name": "pengumuman"}],
            },
            token=token,
        )
        assert st == 200, f"Failed to create welcome question: {st} {qc_resp}"
        print("Seeded welcome Q&A question in Apache Answer")
    else:
        print("Welcome Q&A question already present")


if __name__ == "__main__":
    main()
