import sys

with open('internal/ui/chat/state.go', 'r') as f:
    content = f.read()

search_block = """			v.app.QueueUpdateDraw(func() {
				v.app.Stop()
			})"""

replace_block = """			v.app.QueueUpdateDraw(func() {
				v.showConfirmModal(
					fmt.Sprintf("Failed to connect to Discord:\n%s", err),
					[]string{"Quit"},
					func(_ string) {
						v.app.Stop()
					},
				)
			})"""

if search_block in content:
    content = content.replace(search_block, replace_block)
    with open('internal/ui/chat/state.go', 'w') as f:
        f.write(content)
    print("Successfully replaced block.")
else:
    print("Could not find block to replace.")
    sys.exit(1)
