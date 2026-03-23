package main

func main() {
	initLogger()
	loadConfig()
	loadBotSettings()
	ensureMountStreams()
	startIdleStreamReaper()
	startBot()
}
