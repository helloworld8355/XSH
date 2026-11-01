package builtins

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"xsh/internal/pipeline"
	"xsh/internal/shell"
	"xsh/internal/ui"
)

func registerFS(reg *shell.Registry) {
	reg.Add(&shell.Command{Name: "ls", Aliases: []string{"Get-ChildItem", "dir"}, Summary: "列出目录 [-r递归] [路径]", Fn: cmdLS})
	reg.Add(&shell.Command{Name: "cd", Aliases: []string{"Set-Location"}, Summary: "切换目录 [路径]", Fn: cmdCD})
	reg.Add(&shell.Command{Name: "pwd", Aliases: []string{"Get-Location"}, Summary: "当前目录", Fn: cmdPWD})
	reg.Add(&shell.Command{Name: "cat", Aliases: []string{"Get-Content", "type"}, Summary: "查看文件内容 <文件...>", Fn: cmdCat})
	reg.Add(&shell.Command{Name: "cp", Aliases: []string{"Copy-Item"}, Summary: "复制 [-r] <源> <目标>", Fn: cmdCp})
	reg.Add(&shell.Command{Name: "mv", Aliases: []string{"Move-Item"}, Summary: "移动/重命名 <源> <目标>", Fn: cmdMv})
	reg.Add(&shell.Command{Name: "rm", Aliases: []string{"Remove-Item", "del"}, Summary: "删除 [-r递归] [-f忽略缺失] <路径...>", Fn: cmdRm})
	reg.Add(&shell.Command{Name: "mkdir", Aliases: []string{"New-Item"}, Summary: "创建目录 <目录...>", Fn: cmdMkdir})
	reg.Add(&shell.Command{Name: "tree", Summary: "目录树 [路径] [深度=3]", Fn: cmdTree})
	reg.Add(&shell.Command{Name: "find", Summary: "按名称查找 <模式> [路径]", Fn: cmdFind})
	reg.Add(&shell.Command{Name: "stat", Aliases: []string{"Get-Item"}, Summary: "文件信息 <路径...>", Fn: cmdStat})
	reg.Add(&shell.Command{Name: "du", Summary: "目录体积 [路径]", Fn: cmdDU})
	reg.Add(&shell.Command{Name: "touch", Summary: "创建空文件/更新时间 <文件...>", Fn: cmdTouch})
}

func fileSchema() *pipeline.Schema {
	return pipeline.NewSchema("name", "type", "size", "modified")
}

func mkFileRec(name string, d fs.DirEntry) pipeline.Record {
	typ, size, mod := "file", "", ""
	if fi, err := d.Info(); err == nil {
		mod = fmtTime(fi.ModTime())
		if fi.IsDir() {
			typ = "dir"
		} else {
			size = ui.HumanBytes(float64(fi.Size()))
		}
	}
	return pipeline.Record{"name": name, "type": typ, "size": size, "modified": mod}
}

func cmdLS(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, nil)
	recursive := flags["r"] != "" || flags["R"] != ""
	dir := c.Resolve(first(rest))
	info, err := os.Stat(dir)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		return []pipeline.Record{mkFileRec(filepath.Base(dir), fs.FileInfoToDirEntry(info))}, fileSchema(), nil
	}
	var recs []pipeline.Record
	if recursive {
		count := 0
		root := dir
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipDir
			}
			if p == root {
				return nil
			}
			if count > 5000 {
				return filepath.SkipAll
			}
			count++
			rel, e := filepath.Rel(root, p)
			if e != nil {
				rel = p
			}
			recs = append(recs, mkFileRec(rel, d))
			return nil
		})
		if err != nil && err != filepath.SkipAll {
			return nil, nil, err
		}
		return recs, fileSchema(), nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	for _, e := range entries {
		recs = append(recs, mkFileRec(e.Name(), e))
	}
	return recs, fileSchema(), nil
}

func cmdCD(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		fmt.Fprintln(c.Out, c.Cwd)
		return nil, nil, nil
	}
	p := c.Resolve(rest[0])
	info, err := os.Stat(p)
	if err != nil {
		return nil, nil, err
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("%s 不是目录", p)
	}
	c.Cwd = p
	return nil, nil, nil
}

func cmdPWD(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	return []pipeline.Record{{"path": c.Cwd}}, pipeline.NewSchema("path"), nil
}

func cmdCat(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		if len(in) > 0 {
			return in, pipeline.LineSchema(), nil
		}
		return argErr("cat", "<文件...>")
	}
	var recs []pipeline.Record
	for _, f := range rest {
		p := c.Resolve(f)
		if isBinary(p) {
			return nil, nil, fmt.Errorf("%s: 二进制文件", f)
		}
		lines, err := readFileLines(p, 0)
		if err != nil {
			return nil, nil, err
		}
		for _, ln := range lines {
			recs = append(recs, pipeline.Record{"line": ln})
		}
	}
	return recs, pipeline.LineSchema(), nil
}

func cmdCp(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, nil)
	if len(rest) < 2 {
		return argErr("cp", "[-r] <源...> <目标>")
	}
	dst := c.Resolve(rest[len(rest)-1])
	for _, src := range rest[:len(rest)-1] {
		sp := c.Resolve(src)
		info, err := os.Stat(sp)
		if err != nil {
			return nil, nil, err
		}
		target := dst
		if di, err := os.Stat(dst); err == nil && di.IsDir() {
			target = filepath.Join(dst, filepath.Base(sp))
		}
		if info.IsDir() {
			if flags["r"] == "" && flags["R"] == "" {
				return nil, nil, fmt.Errorf("%s 是目录，需要 -r", sp)
			}
			if err := copyDir(sp, target); err != nil {
				return nil, nil, err
			}
			continue
		}
		if err := copyFile(sp, target); err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, nil
}

func cmdMv(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) < 2 {
		return argErr("mv", "<源...> <目标>")
	}
	dst := c.Resolve(rest[len(rest)-1])
	for _, src := range rest[:len(rest)-1] {
		sp := c.Resolve(src)
		target := dst
		if di, err := os.Stat(dst); err == nil && di.IsDir() {
			target = filepath.Join(dst, filepath.Base(sp))
		}
		if err := os.Rename(sp, target); err != nil {
			// 跨盘回退：复制后删除
			if cpErr := copyPath(sp, target); cpErr != nil {
				return nil, nil, err
			}
			if rmErr := os.RemoveAll(sp); rmErr != nil {
				return nil, nil, rmErr
			}
		}
	}
	return nil, nil, nil
}

func cmdRm(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	flags, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("rm", "[-r] [-f] <路径...>")
	}
	for _, f := range rest {
		p := c.Resolve(f)
		info, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) && flags["f"] != "" {
				continue
			}
			return nil, nil, err
		}
		if info.IsDir() && flags["r"] == "" && flags["R"] == "" {
			return nil, nil, fmt.Errorf("%s 是目录，需要 -r", p)
		}
		if err := os.RemoveAll(p); err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, nil
}

func cmdMkdir(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("mkdir", "<目录...>")
	}
	for _, d := range rest {
		if err := os.MkdirAll(c.Resolve(d), 0o755); err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, nil
}

func cmdTree(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	root := c.Resolve(first(rest))
	depth := 3
	if len(rest) > 1 {
		depth = atoi(rest[1], 3)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, nil, fmt.Errorf("%s 不是有效目录", root)
	}
	fmt.Fprintln(c.Out, root)
	count := 0
	var walk func(dir, prefix string, d int)
	walk = func(dir, prefix string, d int) {
		if d > depth || count > 2000 {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
		})
		for i, e := range entries {
			count++
			connector := "├── "
			if i == len(entries)-1 {
				connector = "└── "
			}
			fmt.Fprintf(c.Out, "%s%s%s\n", prefix, connector, e.Name())
			if e.IsDir() {
				np := prefix + "│   "
				if i == len(entries)-1 {
					np = prefix + "    "
				}
				walk(filepath.Join(dir, e.Name()), np, d+1)
			}
		}
	}
	walk(root, "", 1)
	return nil, nil, nil
}

func cmdFind(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("find", "<模式> [路径]")
	}
	pattern := strings.ToLower(rest[0])
	root := c.Cwd
	if len(rest) > 1 {
		root = c.Resolve(rest[1])
	}
	var recs []pipeline.Record
	count := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if count > 5000 {
			return filepath.SkipAll
		}
		lower := strings.ToLower(d.Name())
		match := strings.Contains(lower, pattern)
		if !match {
			if m, e := filepath.Match(pattern, lower); e == nil && m {
				match = true
			}
		}
		if match {
			count++
			typ, size := "file", ""
			if d.IsDir() {
				typ = "dir"
			} else if fi, e := d.Info(); e == nil {
				size = ui.HumanBytes(float64(fi.Size()))
			}
			recs = append(recs, pipeline.Record{"path": p, "type": typ, "size": size})
		}
		return nil
	})
	if err != nil && err != filepath.SkipAll {
		return nil, nil, err
	}
	return recs, pipeline.NewSchema("path", "type", "size"), nil
}

func cmdStat(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("stat", "<路径...>")
	}
	var recs []pipeline.Record
	sc := pipeline.NewSchema("name", "path", "type", "size", "mode", "modified").WithNumeric("size")
	for _, f := range rest {
		p := c.Resolve(f)
		info, err := os.Stat(p)
		if err != nil {
			return nil, nil, err
		}
		typ := "file"
		if info.IsDir() {
			typ = "dir"
		}
		recs = append(recs, pipeline.Record{
			"name":     info.Name(),
			"path":     p,
			"type":     typ,
			"size":     fmt.Sprintf("%d", info.Size()),
			"mode":     info.Mode().String(),
			"modified": info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}
	return recs, sc, nil
}

func cmdDU(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	root := c.Resolve(first(rest))
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, nil, fmt.Errorf("%s 不是有效目录", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, nil, err
	}
	var recs []pipeline.Record
	var total int64
	var totalFiles int
	for _, e := range entries {
		p := filepath.Join(root, e.Name())
		var size int64
		var files int
		if e.IsDir() {
			size, files = dirSize(p, 3*time.Second)
		} else if fi, err := e.Info(); err == nil {
			size, files = fi.Size(), 1
		}
		total += size
		totalFiles += files
		recs = append(recs, pipeline.Record{
			"name":  e.Name(),
			"size":  ui.HumanBytes(float64(size)),
			"files": fmt.Sprintf("%d", files),
		})
	}
	recs = append(recs, pipeline.Record{
		"name":  "(总计)",
		"size":  ui.HumanBytes(float64(total)),
		"files": fmt.Sprintf("%d", totalFiles),
	})
	sc := pipeline.NewSchema("name", "size", "files").WithNumeric("files")
	pipeline.SortRecords(recs, "size", true, false)
	// 总计保持在最后
	n := len(recs)
	if n > 1 {
		total2 := recs[n-1]
		recs = append(recs[:n-1], recs[n-1:]...)
		recs = append(recs[:len(recs)-1], total2)
	}
	return recs, sc, nil
}

// dirSize 限时统计目录大小。
func dirSize(path string, maxTime time.Duration) (int64, int) {
	deadline := time.Now().Add(maxTime)
	var size int64
	var files int
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			if time.Now().After(deadline) {
				return filepath.SkipAll
			}
			if fi, e := d.Info(); e == nil {
				size += fi.Size()
				files++
			}
		}
		return nil
	})
	return size, files
}

func cmdTouch(c *shell.Ctx, args []string, in []pipeline.Record, inSc *pipeline.Schema) ([]pipeline.Record, *pipeline.Schema, error) {
	_, rest := parseArgs(args, nil)
	if len(rest) == 0 {
		return argErr("touch", "<文件...>")
	}
	now := time.Now()
	for _, f := range rest {
		p := c.Resolve(f)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			fh, err := os.Create(p)
			if err != nil {
				return nil, nil, err
			}
			fh.Close()
		} else if err := os.Chtimes(p, now, now); err != nil {
			return nil, nil, err
		}
	}
	return nil, nil, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(p, target)
	})
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}
