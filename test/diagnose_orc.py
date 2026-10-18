import UnityPy

assets_path = r'D:\games\Orc Massage\OrcMassage_Data\resources.assets'
print(f"Loading {assets_path}...")
env = UnityPy.load(assets_path)
print(f"Loaded! Total objects: {len(env.objects)}")

search_phrases = ['Whaddya want', 'Employment contract', 'VIP Massage Plan', 'Massage Techniques', 'Thalia']

found = 0
for obj in env.objects:
    try:
        raw_bytes = obj.get_raw_data()
    except Exception:
        continue

    for phrase in search_phrases:
        if phrase.encode('utf-8') in raw_bytes:
            found += 1
            print(f"\n--- Found '{phrase}' in obj path_id={obj.path_id}, type={obj.type.name} ---")
            
            # Check serialized type
            st = getattr(obj, "serialized_type", None)
            has_nodes = False
            if st and getattr(st, "nodes", None):
                has_nodes = True
            print(f"  has_typetree_nodes: {has_nodes}")

            # Try read()
            try:
                data = obj.read()
                print(f"  read() success: class={type(data).__name__}")
                if hasattr(data, "m_Name"):
                    print(f"  m_Name={data.m_Name}")
            except Exception as e:
                print(f"  read() error: {e}")

            # Try read_typetree()
            try:
                tree = obj.read_typetree()
                print(f"  read_typetree() success!")
                # Let's find where the phrase appears in tree
                def search_tree(val, p):
                    if isinstance(val, str):
                        if phrase in val:
                            print(f"    FOUND in tree at path: {p} => {val!r}")
                    elif isinstance(val, dict):
                        for k, v in val.items():
                            search_tree(v, f"{p}.{k}" if p else k)
                    elif isinstance(val, list):
                        for i, item in enumerate(val):
                            search_tree(item, f"{p}[{i}]")
                search_tree(tree, "root")
            except Exception as e:
                print(f"  read_typetree() error: {e}")

            break

    if found >= 10:
        break

print(f"\nTotal matches found: {found}")
