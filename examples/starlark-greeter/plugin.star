def run(input):
    ts = time_now_unix_ms()
    log("starlark >> input received")
    emit_event("greeting.created", "input=" + input + ";at_ms=" + str(ts))
    return "Hello, " + input
