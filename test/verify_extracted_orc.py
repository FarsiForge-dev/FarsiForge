import json

with open(r'D:\FarsiForge-dev\test_extract_orc\extracted.json', 'r', encoding='utf-8') as f:
    entries = json.load(f)

print(f"Total loaded entries: {len(entries)}")

search_phrases = [
    "Whaddya want? Come in.",
    "Employment contract",
    "START GAME",
    "VIP Massage Plan",
    "Massage Techniques",
    "Thalia",
    "Mystery Reel Episode 2",
    "VIP Name",
    "VIP Status",
    "Shoulders",
    "Options",
    "Video",
    "Sound",
    "Language",
    "Unread Message",
    "Acquired new knowledge"
]

found = {}
for e in entries:
    src = e.get('source', '')
    cat = e.get('category', '')
    cls = e.get('candidate_class', '')
    path = e.get('path', '')
    for p in search_phrases:
        if p.lower() in src.lower():
            found.setdefault(p, []).append((cat, cls, path, src))

print('\n' + '='*60)
for p in search_phrases:
    matches = found.get(p, [])
    if matches:
        m = matches[0]
        print(f"[FOUND] \"{p}\": {len(matches)} occurrences")
        print(f"   Category: {m[0]} | Class: {m[1]} | Path: {m[2]}")
        print(f"   Sample: {m[3]!r}")
    else:
        print(f"[NOT FOUND] \"{p}\"")
print('='*60)
