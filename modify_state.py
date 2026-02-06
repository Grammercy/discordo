import sys

path = "internal/ui/chat/state.go"
with open(path, "r") as f:
    content = f.read()

old_code = """	v.state.OnRequest = append(v.state.OnRequest, httputil.WithHeaders(http.Headers()), v.onRequest)
	return v.state.Open(context.TODO())
}"""

new_code = """	v.state.OnRequest = append(v.state.OnRequest, httputil.WithHeaders(http.Headers()), v.onRequest)

	go func() {
		if err := v.state.Open(context.TODO()); err != nil {
			slog.Error("failed to open state", "err", err)
			v.app.QueueUpdateDraw(func() {
				v.app.Stop()
			})
		}
	}()

	return nil
}"""

if old_code in content:
    content = content.replace(old_code, new_code)
    with open(path, "w") as f:
        f.write(content)
    print("Successfully modified state.go")
else:
    print("Could not find the code block to replace")
    sys.exit(1)
