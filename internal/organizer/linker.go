package organizer

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// TransferFile 根据指定的模式 (硬链接/复制/移动/软链接) 执行文件落地
func TransferFile(src, dst string, mode LinkMode) error {
	// 确保目标所在父目录存在
	targetDir := filepath.Dir(dst)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target dir error: %w", err)
	}

	// 检查目标文件是否已存在
	if dstFi, err := os.Stat(dst); err == nil {
		srcFi, err := os.Stat(src)
		if err == nil && os.SameFile(srcFi, dstFi) {
			// 源文件与目标文件完全相同 (已经是硬链接)，无需重复处理
			return nil
		}
		// 目标已存在但非同一文件，先移除已有目标
		_ = os.Remove(dst)
	}

	switch mode {
	case LinkModeHardlink:
		err := os.Link(src, dst)
		if err != nil {
			// 跨盘/跨分区时 os.Link 会报错 (EXDEV: cross-device link)
			slog.Warn("Hardlink failed (possible cross-device/partition), falling back to copy", "src", src, "dst", dst, "error", err)
			return copyFile(src, dst)
		}
		return nil

	case LinkModeSymlink:
		return os.Symlink(src, dst)

	case LinkModeMove:
		err := os.Rename(src, dst)
		if err != nil {
			// 跨盘移动失败时 fallback: 复制后删除原文件
			if copyErr := copyFile(src, dst); copyErr != nil {
				return copyErr
			}
			return os.Remove(src)
		}
		return nil

	case LinkModeCopy:
		return copyFile(src, dst)

	default:
		return fmt.Errorf("unsupported link mode: %s", mode)
	}
}

// copyFile 物理文件流复制
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src file error: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst file error: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy file stream error: %w", err)
	}

	return out.Sync()
}
