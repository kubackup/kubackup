package model

// StatusUpdate 进度
type StatusUpdate struct {
	MessageType      string   `json:"messageType"`      // "status"
	SecondsElapsed   string   `json:"secondsElapsed"`   // 已用时间
	SecondsRemaining string   `json:"secondsRemaining"` // 剩余时间
	PercentDone      float64  `json:"percentDone"`      // 进度
	TotalFiles       uint64   `json:"totalFiles"`       // 文件总数
	FilesDone        uint64   `json:"filesDone"`        // 完成文件数
	TotalBytes       string   `json:"totalBytes"`       // 文件总大小
	BytesDone        string   `json:"bytesDone"`        // 完成文件大小
	ErrorCount       uint     `json:"errorCount"`       // 错误数量
	CurrentFiles     []string `json:"currentFiles"`     // 当前文件列表
}

// ErrorUpdate 错误
type ErrorUpdate struct {
	MessageType string `json:"messageType"` // "error"
	Error       string `json:"error"`       // 错误信息
	During      string `json:"during"`      // 错误类型
	Item        string `json:"item"`        // 错误目标
}

// VerboseUpdate 完成项
type VerboseUpdate struct {
	MessageType  string `json:"messageType"` // "verbose_status"
	Action       string `json:"action"`
	Item         string `json:"item"`
	Duration     string `json:"duration"`     // 持续时间 in seconds
	DataSize     string `json:"dataSize"`     // 数据量
	MetadataSize string `json:"metadataSize"` // 元数据量
	TotalFiles   uint   `json:"totalFiles"`   // 总文件数
}

// SummaryOutput 备份完成汇总
type SummaryOutput struct {
	MessageType         string `json:"messageType"` // "summary"
	FilesNew            uint   `json:"filesNew"`
	FilesChanged        uint   `json:"filesChanged"`
	FilesUnmodified     uint   `json:"filesUnmodified"`
	DirsNew             uint   `json:"dirsNew"`
	DirsChanged         uint   `json:"dirsChanged"`
	DirsUnmodified      uint   `json:"dirsUnmodified"`
	DataBlobs           int    `json:"dataBlobs"`
	TreeBlobs           int    `json:"treeBlobs"`
	DataAdded           string `json:"dataAdded"`            //本次新增文件大小
	TotalFilesProcessed uint   `json:"totalFiles_processed"` // 文件总数
	TotalBytesProcessed string `json:"totalBytes_processed"` // 文件总大小
	TotalDuration       string `json:"totalDuration"`        // 总耗时 in seconds
	SnapshotID          string `json:"snapshotId"`
	DryRun              bool   `json:"dryRun,omitempty"` //
}
