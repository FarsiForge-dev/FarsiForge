#!/usr/bin/env python3
"""
FarsiForge Telegram AI Notifier
Automatically monitors git commits and releases across FarsiForge repositories,
summarizes changes using AI (Gemini / OpenAI), and posts high-impact updates
directly to Telegram channels and groups.
"""

import os
import sys
import json
import re
import subprocess
import urllib.request
import urllib.parse
from typing import List, Dict, Optional, Tuple, Any

# Ensure UTF-8 encoding on all systems (especially Windows runners)
if hasattr(sys.stdout, "reconfigure"):
    try:
        sys.stdout.reconfigure(encoding="utf-8")
        sys.stderr.reconfigure(encoding="utf-8")
    except Exception:
        pass


def parse_github_event(event_path: str) -> Optional[Dict[str, Any]]:
    """Parse GitHub Actions event payload JSON if available."""
    if not event_path or not os.path.exists(event_path):
        return None
    try:
        with open(event_path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception as e:
        print(f"[WARN] Failed to parse GITHUB_EVENT_PATH: {e}", file=sys.stderr)
        return None


def get_commit_data_from_git(max_commits: int = 15) -> Dict[str, Any]:
    """Fallback: extract branch, author, and recent commits directly using git CLI."""
    data = {
        "ref_name": "main",
        "is_tag": False,
        "author": "تیم توسعه",
        "compare_url": "",
        "commits": []
    }
    
    # Detect current branch or tag
    try:
        ref = subprocess.check_output(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            stderr=subprocess.DEVNULL,
            text=True
        ).strip()
        if ref == "HEAD":
            # Might be detached HEAD (tag)
            tag = subprocess.check_output(
                ["git", "describe", "--tags", "--exact-match"],
                stderr=subprocess.DEVNULL,
                text=True
            ).strip()
            data["ref_name"] = tag if tag else "HEAD"
            data["is_tag"] = bool(tag)
        else:
            data["ref_name"] = ref
    except Exception:
        pass

    # Extract recent commits: short_hash|author_name|subject
    try:
        raw_log = subprocess.check_output(
            ["git", "log", f"-n{max_commits}", "--pretty=format:%h|%an|%s"],
            stderr=subprocess.DEVNULL,
            text=True
        ).strip().splitlines()

        for line in raw_log:
            if not line.strip():
                continue
            parts = line.split("|", 2)
            if len(parts) == 3:
                h, author, msg = parts
                data["commits"].append({
                    "id": h.strip(),
                    "author": author.strip(),
                    "message": msg.strip()
                })
        if data["commits"]:
            data["author"] = data["commits"][0]["author"]
    except Exception as e:
        data["commits"] = [{"id": "latest", "author": "توسعه‌دهنده", "message": f"Update: {e}"}]

    return data


def extract_event_details() -> Dict[str, Any]:
    """Extract push / commit / tag metadata from GitHub event or git environment."""
    event_path = os.environ.get("GITHUB_EVENT_PATH", "")
    payload = parse_github_event(event_path)
    
    repo_full = os.environ.get("GITHUB_REPOSITORY", "FarsiForge-dev/FarsiForge")
    repo_name = repo_full.split("/")[-1]

    if payload:
        ref = payload.get("ref", "")
        is_tag = ref.startswith("refs/tags/")
        if is_tag:
            ref_name = ref.replace("refs/tags/", "")
        elif ref.startswith("refs/heads/"):
            ref_name = ref.replace("refs/heads/", "")
        else:
            ref_name = os.environ.get("GITHUB_REF_NAME", "main")

        pusher = (
            payload.get("pusher", {}).get("name")
            or payload.get("sender", {}).get("login")
            or "توسعه‌دهنده"
        )
        compare_url = payload.get("compare", f"https://github.com/{repo_full}")

        commits_raw = payload.get("commits", [])
        commits = []
        for c in commits_raw:
            commits.append({
                "id": c.get("id", "")[:7],
                "author": c.get("author", {}).get("name", pusher),
                "message": c.get("message", "").strip(),
                "url": c.get("url", "")
            })

        # If payload had no commits list (e.g. tag pushed directly or merge), fall back to git log
        if not commits:
            git_data = get_commit_data_from_git()
            commits = git_data["commits"]

        return {
            "repo_name": repo_name,
            "repo_full": repo_full,
            "ref_name": ref_name,
            "is_tag": is_tag,
            "pusher": pusher,
            "compare_url": compare_url,
            "commits": commits
        }

    # Running outside GitHub Actions or no event payload
    git_data = get_commit_data_from_git()
    return {
        "repo_name": repo_name,
        "repo_full": repo_full,
        "ref_name": git_data["ref_name"],
        "is_tag": git_data["is_tag"],
        "pusher": git_data["author"],
        "compare_url": f"https://github.com/{repo_full}",
        "commits": git_data["commits"]
    }


def call_gemini(api_key: str, prompt: str) -> Optional[str]:
    """Query Google Gemini API for fast, natural Persian AI summaries."""
    models = ["gemini-2.5-flash", "gemini-flash-latest", "gemini-3.8-flash", "gemini-2.0-flash", "gemini-1.5-flash"]
    for model in models:
        m = model.replace("models/", "")
        url = f"https://generativelanguage.googleapis.com/v1beta/models/{m}:generateContent?key={api_key}"
        payload = {
            "contents": [{"parts": [{"text": prompt}]}],
            "generationConfig": {
                "temperature": 0.5,
                "maxOutputTokens": 1200
            }
        }
        data = json.dumps(payload).encode("utf-8")
        req = urllib.request.Request(
            url,
            data=data,
            headers={"Content-Type": "application/json"}
        )
        try:
            with urllib.request.urlopen(req, timeout=8) as resp:
                body = json.loads(resp.read().decode("utf-8"))
                return body["candidates"][0]["content"]["parts"][0]["text"].strip()
        except Exception as err:
            continue
    print("[WARN] All Gemini API endpoints failed or key invalid.", file=sys.stderr)
    return None


def call_openai(api_key: str, prompt: str) -> Optional[str]:
    """Query OpenAI API (gpt-4o-mini) as fallback AI provider."""
    url = "https://api.openai.com/v1/chat/completions"
    payload = {
        "model": "gpt-4o-mini",
        "messages": [
            {
                "role": "system",
                "content": "You are a professional Persian tech communicator and community manager for FarsiForge."
            },
            {"role": "user", "content": prompt}
        ],
        "temperature": 0.5,
        "max_tokens": 1200
    }
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {api_key}"
        }
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            body = json.loads(resp.read().decode("utf-8"))
            return body["choices"][0]["message"]["content"].strip()
    except Exception as err:
        print(f"[WARN] OpenAI API call failed: {err}", file=sys.stderr)
        return None


def clean_html_for_telegram(text: str) -> str:
    """Sanitize output to ensure Telegram HTML parser compatibility."""
    # Strip markdown code blocks
    text = re.sub(r"^```(?:html)?\s*", "", text, flags=re.IGNORECASE)
    text = re.sub(r"\s*```$", "", text)
    # Telegram only accepts <b>, <i>, <u>, <s>, <a>, <code>, <pre>, <blockquote>
    # Remove unsupported HTML tags while preserving text
    text = re.sub(r"</?(?:div|p|span|h[1-6]|ul|ol|li|br)\b[^>]*>", "\n", text, flags=re.IGNORECASE)
    # Clean multiple consecutive blank lines
    text = re.sub(r"\n{3,}", "\n\n", text)
    return text.strip()


def build_fallback_message(info: Dict[str, Any]) -> str:
    """Deterministic, high-quality Persian message when AI is unavailable."""
    repo = info["repo_name"]
    ref = info["ref_name"]
    pusher = info["pusher"]
    commits = info["commits"]
    url = info["compare_url"]
    is_tag = info["is_tag"]

    if is_tag:
        header = f"🎮 <b>آپدیت و نسخه جدید {repo} منتشر شد!</b>\n🏷 <b>نسخه:</b> <code>{ref}</code> | 👤 <b>توسعه‌دهنده:</b> {pusher}"
    else:
        header = f"🛠 <b>تغییرات و کامیت‌های جدید در {repo}</b>\n🌿 <b>شاخه:</b> <code>{ref}</code> | 👤 <b>توسعه‌دهنده:</b> {pusher}"

    features = []
    fixes = []
    others = []

    for c in commits:
        msg = c["message"].split("\n")[0]  # First line of commit
        cid = c["id"]
        formatted_c = f"• <code>{cid}</code> {msg}"
        low = msg.lower()
        if low.startswith("feat") or "add" in low or "feature" in low:
            features.append(formatted_c)
        elif low.startswith("fix") or "bug" in low or "patch" in low:
            fixes.append(formatted_c)
        else:
            others.append(formatted_c)

    sections = [header, ""]

    if features:
        sections.append("✨ <b>قابلیت‌های جدید:</b>")
        sections.extend(features[:5])
        sections.append("")

    if fixes:
        sections.append("🛠 <b>رفع مشکلات و بهینه‌سازی:</b>")
        sections.extend(fixes[:5])
        sections.append("")

    if others and not features and not fixes:
        sections.append("📋 <b>لیست تغییرات:</b>")
        sections.extend(others[:6])
        sections.append("")
    elif others and (features or fixes):
        sections.append("⚡ <b>سایر تغییرات:</b>")
        sections.extend(others[:3])
        sections.append("")

    sections.append(f"🔗 <a href=\"{url}\">مشاهده تغییرات و جزئیات در گیت‌هاب</a>")
    sections.append("")
    sections.append("#FarsiForge #آپدیت #توسعه #GameLocalization")

    return "\n".join(sections)


def send_telegram_message(bot_token: str, chat_id: str, html_text: str) -> bool:
    """Publish the formatted message to Telegram Bot API."""
    url = f"https://api.telegram.org/bot{bot_token}/sendMessage"
    payload = {
        "chat_id": chat_id,
        "text": html_text,
        "parse_mode": "HTML",
        "disable_web_page_preview": False
    }
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url,
        data=data,
        headers={"Content-Type": "application/json"}
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            result = json.loads(resp.read().decode("utf-8"))
            if result.get("ok"):
                print("[SUCCESS] Telegram notification posted successfully!")
                return True
            else:
                print(f"[ERROR] Telegram returned error: {result}", file=sys.stderr)
                return False
    except urllib.error.HTTPError as http_err:
        err_content = http_err.read().decode("utf-8", errors="ignore")
        print(f"[ERROR] Telegram HTTP {http_err.code}: {err_content}", file=sys.stderr)
        return False
    except Exception as e:
        print(f"[ERROR] Could not connect to Telegram: {e}", file=sys.stderr)
        return False


def main():
    bot_token = os.environ.get("TELEGRAM_BOT_TOKEN", "").strip()
    chat_id = os.environ.get("TELEGRAM_CHAT_ID", "").strip()
    gemini_key = os.environ.get("GEMINI_API_KEY", "").strip()
    openai_key = os.environ.get("OPENAI_API_KEY", "").strip()

    info = extract_event_details()
    commits = info["commits"]

    if not commits:
        print("[INFO] No commits found in this push event. Skipping Telegram post.")
        return

    repo_name = info["repo_name"]
    ref_name = info["ref_name"]
    is_tag = info["is_tag"]
    pusher = info["pusher"]
    url = info["compare_url"]

    print(f"[INFO] Processing {len(commits)} commits for {repo_name} ({ref_name}) by {pusher}...")

    # Build commit digest for AI
    commit_lines = []
    for c in commits[:20]:
        first_line = c["message"].split("\n")[0]
        commit_lines.append(f"- [{c['id']}] {first_line} (by {c.get('author', pusher)})")
    commits_text = "\n".join(commit_lines)

    context_type = "انتشار نسخه جدید و تگ" if is_tag else "پوش کامیت‌ها و تغییرات سورس کد"

    ai_prompt = f"""
تو مدیر فنی و مدیر کامیونیتی پروژه متن‌باز «FarsiForge» (توسعه و فارسی‌سازی بازی‌های ویدیویی) هستی.
یک پست تلگرام جذاب، مرتب، به زبان فارسی و با لحن حرفه‌ای و صمیمی درباره {context_type} آماده کن.

مشخصات رویداد:
- مخزن / ابزار: {repo_name}
- {'نسخه' if is_tag else 'شاخه'}: {ref_name}
- توسعه‌دهنده: {pusher}
- لینک مشاهده در گیت‌هاب: {url}

لیست کامیت‌ها:
{commits_text}

دستورالعمل‌ها:
1. کامیت‌های فنی و انگلیسی را تحلیل، درک و ترجمه کن و به زبان فارسی روان توضیح بده.
2. بر اساس محتوای کامیت‌ها، بخش‌بندی کن:
   - ✨ قابلیت‌ها و ویژگی‌های جدید (در صورت وجود)
   - ⚡ بهبودها و بهینه‌سازی‌ها
   - 🛠 رفع مشکلات و اصلاحات (در صورت وجود)
3. نام مخزن ({repo_name})، {'نسخه' if is_tag else 'شاخه'} ({ref_name}) و توسعه‌دهنده ({pusher}) را در بالای پست با ایموجی زیبا درج کن.
4. در انتهای پست لینک مشاهده در گیت‌هاب را به صورت هایپرلینک HTML قرار بده: <a href="{url}">مشاهده تغییرات در گیت‌هاب</a>
5. حتماً از تگ‌های استاندارد HTML مجاز تلگرام استفاده کن (فقط و فقط <b>، <i>، <code>، <a href="...">). هرگز از مارک‌داون یا تگ‌های دیگر استفاده نکن.
6. در انتهای پست چند هشتگ مرتبط بگذار (#FarsiForge #{repo_name} #توسعه #آپدیت).
7. خروجی فقط و فقط متن نهایی پیام باشد، بدون هیچ توضیح اضافی یا بلاک کد ```.
"""

    post_content = None

    if gemini_key:
        print("[INFO] Contacting Gemini AI for Persian announcement generation...")
        post_content = call_gemini(gemini_key, ai_prompt)

    if not post_content and openai_key:
        print("[INFO] Contacting OpenAI for Persian announcement generation...")
        post_content = call_openai(openai_key, ai_prompt)

    if not post_content:
        print("[INFO] Using deterministic Persian template fallback...")
        post_content = build_fallback_message(info)

    post_content = clean_html_for_telegram(post_content)

    # Dry-run or missing secrets check
    is_dry_run = "--dry-run" in sys.argv or not bot_token or not chat_id
    if is_dry_run:
        print("\n==================== [TELEGRAM MESSAGE PREVIEW] ====================")
        print(post_content)
        print("=====================================================================\n")
        if not bot_token or not chat_id:
            print("[NOTICE] TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID is missing from environment.")
            print("[NOTICE] Displayed output above. Once secrets are added to the repo/org, it will post directly!")
        return

    # Broadcast to Telegram
    send_telegram_message(bot_token, chat_id, post_content)


if __name__ == "__main__":
    main()
