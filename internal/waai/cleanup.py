import sys

with open('internal/waai/ai.go', 'r') as f:
    lines = f.readlines()

# Find the BuildCompanySystemPrompt return and clean up garbage
# After line 1306 (0-indexed 1305), we should have just the closing
# The correct ending is: after `currentMonth)` at line 1306, we need:
#
# 	return prompt
# }
#
# And lines 1307+ are garbage that need to be removed.

# Find where "return prompt" and "}" are
for i, line in enumerate(lines):
    stripped = line.strip()
    # Find the legitimate ending of BuildCompanySystemPrompt
    if i > 1300 and stripped.startswith('return prompt'):
        # This is the real one - everything before 1306 should be kept as-is
        break

# The garbage starts at line 1307 (the \n after currentMonth) and continues
# until we find the real 'return prompt' + '}'
# Strategy: take lines[0:1307], then find the REAL ending after the garbage

# Find the first legitimate 'return prompt' after line 1300
real_return = -1
for i in range(1300, len(lines)):
    if lines[i].strip() == 'return prompt':
        real_return = i
        break

if real_return >= 0:
    # Keep up to 1306, then append from real_return onward
    new_lines = lines[:1307] + lines[real_return:]
    with open('internal/waai/ai.go', 'w') as f:
        f.writelines(new_lines)
    print(f'SUCCESS: Removed garbage lines 1307-{real_return-1}')
else:
    print('ERROR: Could not find return prompt')
