import sys

with open('internal/ui/chat/state.go', 'r') as f:
    content = f.read()

# The broken block has a literal newline inside the string
broken_block = """					fmt.Sprintf("Failed to connect to Discord:
%s", err),"""

# The correct block should have \n inside the string
correct_block = """					fmt.Sprintf("Failed to connect to Discord:\n%s", err),"""

if broken_block in content:
    content = content.replace(broken_block, correct_block)
    with open('internal/ui/chat/state.go', 'w') as f:
        f.write(content)
    print("Successfully fixed block.")
else:
    print("Could not find broken block to fix.")
    # Maybe check if it's already correct or slightly different
    print("Content around expected location:")
    start_index = content.find("fmt.Sprintf")
    if start_index != -1:
        print(content[start_index:start_index+100])
    sys.exit(1)
