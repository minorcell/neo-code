package workspace

import agentsession "neo-code/internal/session"

// SelectSessionWorkdir 优先返回会话工作目录，缺失时回退到默认工作目录。
func SelectSessionWorkdir(sessionWorkdir string, defaultWorkdir string) string {
	return agentsession.EffectiveWorkdir(sessionWorkdir, defaultWorkdir)
}
