package lineage

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func PrepareRepositoryArchive(ctx context.Context, cfg Config) (string, func(), error) {
	if cfg.BuildSourceDir == "" {
		return "", nil, fmt.Errorf("BUILD_SOURCE_DIR is required")
	}
	baseDir := cfg.LocalArtifactDir
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create local artifact dir: %w", err)
	}
	stagingDir, err := os.MkdirTemp(baseDir, "repo-staging-")
	if err != nil {
		return "", nil, fmt.Errorf("create staging dir: %w", err)
	}
	suffix, err := randomSuffix()
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", nil, err
	}
	archivePath := filepath.Join(baseDir, fmt.Sprintf("lineage-repo-%s.tar.gz", suffix))
	cleanup := func() {
		_ = os.RemoveAll(stagingDir)
		_ = os.Remove(archivePath)
	}

	// [DIAGNOSE] 复制前：打印 sourceDir 内容
	log.Printf("[DIAGNOSE] Pre-copy: listing sourceDir=%s", cfg.BuildSourceDir)
	if listErr := listDirectory(ctx, cfg.BuildSourceDir); listErr != nil {
		log.Printf("[DIAGNOSE] Warning: failed to list sourceDir: %v", listErr)
	}

	if err := stageSourceDirectory(ctx, cfg.BuildSourceDir, stagingDir); err != nil {
		cleanup()
		return "", nil, err
	}

	// [DIAGNOSE] 复制后：打印 dest 内容
	log.Printf("[DIAGNOSE] Post-copy: listing stagingDir=%s", stagingDir)
	if listErr := listDirectory(ctx, stagingDir); listErr != nil {
		log.Printf("[DIAGNOSE] Warning: failed to list stagingDir: %v", listErr)
	}

	// [DIAGNOSE] 压缩前：打印即将打包的路径
	log.Printf("[DIAGNOSE] Pre-archive: creating tar.gz from stagingDir=%s to archivePath=%s", stagingDir, archivePath)

	if err := createRepoArchive(ctx, stagingDir, archivePath); err != nil {
		cleanup()
		return "", nil, err
	}

	// [DIAGNOSE] 压缩后：打印 archive 文件大小和内容列表
	log.Printf("[DIAGNOSE] Post-archive: checking archive=%s", archivePath)
	if statErr := statFile(archivePath); statErr != nil {
		log.Printf("[DIAGNOSE] Warning: failed to stat archive: %v", statErr)
	}
	if listErr := listTarContents(ctx, archivePath); listErr != nil {
		log.Printf("[DIAGNOSE] Warning: failed to list tar contents: %v", listErr)
	}

	return archivePath, cleanup, nil
}

func createRepoArchive(ctx context.Context, sourceDir, archivePath string) error {
	return runLocalCommand(ctx, "tar", "-czf", archivePath, "-C", sourceDir, ".")
}

func runLocalCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message != "" {
			return fmt.Errorf("%s failed: %w: %s", name, err, message)
		}
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func stageSourceDirectory(ctx context.Context, sourceDir, dest string) error {
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("check BUILD_SOURCE_DIR: %w", err)
	}
	// 使用 -T 选项避免创建多余的子目录
	// cp -aT 会把 sourceDir 的内容复制到 dest，不会在 dest 下创建 sourceDir 的子目录
	return runLocalCommand(ctx, "cp", "-aT", sourceDir, dest)
}

// [DIAGNOSE] listDirectory 打印目录的文件列表
func listDirectory(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "ls", "-la", dir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	log.Printf("[DIAGNOSE] Directory listing of %s:\n%s", dir, string(output))

	// 同时打印 find 结果查看完整树形结构
	cmd2 := exec.CommandContext(ctx, "find", dir, "-type", "f")
	output2, err2 := cmd2.CombinedOutput()
	if err2 == nil {
		log.Printf("[DIAGNOSE] File tree of %s:\n%s", dir, string(output2))
	}
	return nil
}

// [DIAGNOSE] statFile 打印文件信息
func statFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	log.Printf("[DIAGNOSE] Archive file: path=%s, size=%d bytes", path, info.Size())
	return nil
}

// [DIAGNOSE] listTarContents 打印 tar.gz 内容列表
func listTarContents(ctx context.Context, archivePath string) error {
	cmd := exec.CommandContext(ctx, "tar", "-tzf", archivePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	log.Printf("[DIAGNOSE] Archive contents:\n%s", string(output))
	return nil
}
