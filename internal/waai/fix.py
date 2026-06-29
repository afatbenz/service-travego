with open('internal/waai/ai.go', 'rb') as f:
    data = f.read()

# The garbage is lines 1307-1321 (0-indexed)
# Line 1306 is the last good line before garbage
# Line 1322+ is good

# Find byte positions
lines = data.split(b'\n')
if len(lines) > 1324:
    # Keep lines 0-1306 (1307 items), then lines 1322+ (from 1322 onward)
    good = lines[:1307] + lines[1322:]
    result = b'\n'.join(good)
    with open('internal/waai/ai.go', 'wb') as f:
        f.write(result)
    print(f'SUCCESS: {len(lines)} -> {len(good)} lines')
else:
    print(f'ERROR: file has {len(lines)} lines, expected > 1324')
