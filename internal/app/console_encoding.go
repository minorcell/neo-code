package app

const utf8CodePage = 65001

var (
	setConsoleOutputCodePage = platformSetConsoleOutputCodePage
	setConsoleInputCodePage  = platformSetConsoleInputCodePage
)

// ensureConsoleUTF8 is best-effort and should never block app startup.
func ensureConsoleUTF8() {
	if err := setConsoleOutputCodePage(utf8CodePage); err != nil {
		return
	}
	_ = setConsoleInputCodePage(utf8CodePage)
}
