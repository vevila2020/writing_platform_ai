package engine

import (
	agenthistory "github.com/vevila2020/writing_platform_ai/agent/context/history"
)

func transcriptVersion(archive *agenthistory.HistoryArchive) uint16 {
	if archive != nil {
		return 2
	}
	return engineTranscriptVersion
}
