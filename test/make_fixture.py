import struct
import UnityPy
from UnityPy.streams import EndianBinaryWriter

def align4(b):
    return b + b'\x00' * ((-len(b)) % 4)

def serialize_string(s):
    if isinstance(s, str):
        b = s.encode('utf-8')
    else:
        b = s
    return align4(struct.pack('<I', len(b)) + b)

def build_test_assets_file():
    # Object 1: Structured JSON TextAsset (dialogue + UI)
    json_text = (
        '{\n'
        '  "dialogues": [\n'
        '    {"speaker": "Arthur", "text": "Welcome to our tavern, traveler! Have a drink."},\n'
        '    {"speaker": "Merchant", "text": "Looking for rare wares? I have what you need."}\n'
        '  ],\n'
        '  "ui": {\n'
        '    "ok": "OK",\n'
        '    "cancel": "Cancel",\n'
        '    "start": "Start Game"\n'
        '  }\n'
        '}'
    )
    ta1_data = serialize_string("localization_data") + serialize_string(json_text)

    # Object 2: Binary / Shader TextAsset (raw heuristic, read-only)
    binary_bytes = b'\x00\x01\x02\xff\xfe\xfd\x80\x90\x00'
    ta2_data = serialize_string("compiled_shaders") + serialize_string(binary_bytes)

    # Metadata and Data are Little Endian (<)
    meta_writer = EndianBinaryWriter(endian="<")
    meta_writer.write_string_to_null('2019.4.1f1')
    meta_writer.write_int(19)  # target_platform = StandaloneWindows64
    meta_writer.write_boolean(False)  # enable_type_tree = False

    # 1 type: TextAsset (class_id = 49)
    meta_writer.write_int(1)
    meta_writer.write_int(49)
    meta_writer.write_boolean(False)
    meta_writer.write_short(-1)
    meta_writer.write_bytes(b'\x00' * 16)

    # 2 objects
    data_writer = EndianBinaryWriter(endian="<")

    # obj 1
    offset1 = data_writer.Length
    data_writer.write_bytes(ta1_data)
    len1 = len(ta1_data)
    data_writer.align_stream(8)

    # obj 2
    offset2 = data_writer.Length
    data_writer.write_bytes(ta2_data)
    len2 = len(ta2_data)
    data_writer.align_stream(8)

    meta_writer.write_int(2)  # object_count
    # obj 1
    meta_writer.align_stream()
    meta_writer.write_long(1001)
    meta_writer.write_u_int(offset1)
    meta_writer.write_u_int(len1)
    meta_writer.write_int(0)

    # obj 2
    meta_writer.align_stream()
    meta_writer.write_long(1002)
    meta_writer.write_u_int(offset2)
    meta_writer.write_u_int(len2)
    meta_writer.write_int(0)

    meta_writer.write_int(0)  # scripts
    meta_writer.write_int(0)  # externals
    meta_writer.write_string_to_null('')  # userInformation

    meta_bytes = meta_writer.bytes
    data_bytes = data_writer.bytes

    header_size = 20  # 4 ints (16) + 1 bool + 3 reserved
    metadata_size = len(meta_bytes)
    data_offset = header_size + metadata_size
    data_offset += (16 - data_offset % 16) % 16
    file_size = data_offset + len(data_bytes)

    # Header is Big Endian (>)
    writer = EndianBinaryWriter(endian=">")
    writer.write_u_int(metadata_size)
    writer.write_u_int(file_size)
    writer.write_u_int(17)
    writer.write_u_int(data_offset)
    writer.write_boolean(False)  # endian = "<"
    writer.write_bytes(b'\x00\x00\x00')

    writer.write_bytes(meta_bytes)
    writer.align_stream(16)
    writer.write_bytes(data_bytes)

    return writer.bytes

if __name__ == '__main__':
    import sys
    out_path = sys.argv[1] if len(sys.argv) > 1 else 'test_fixture.assets'
    raw = build_test_assets_file()
    with open(out_path, 'wb') as f:
        f.write(raw)
    env = UnityPy.load(out_path)
    print(f'Successfully loaded {len(env.objects)} objects to {out_path}!')
    for obj in env.objects:
        d = obj.read()
        name = getattr(d, 'name', getattr(d, 'm_Name', 'unknown'))
        script = getattr(d, 'script', getattr(d, 'm_Script', b''))
        print(f'Object {obj.path_id}: name={name}, len={len(script)}')
