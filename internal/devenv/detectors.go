package devenv

import "regexp"

// detector 工具探测器定义；re 为版本号提取正则（nil 用默认）。
type detector struct {
	name string
	args []string
	re   *regexp.Regexp
	env  []string
}

var (
	reGo      = regexp.MustCompile(`go version go(\S+)`)
	reNode    = regexp.MustCompile(`v(\S+)`)
	reJava    = regexp.MustCompile(`version "?([\w.\-+]+)"?`)
	reKubectl = regexp.MustCompile(`GitVersion:"v?([\w.\-]+)"`)
	reDocker  = regexp.MustCompile(`Docker version (\S+)`)
	reGit     = regexp.MustCompile(`git version (\S+)`)
)

// detectors 全部探测器（顺序即输出顺序）。
var detectors = []detector{
	{name: "git", args: []string{"--version"}, re: reGit, env: []string{"GIT_SSH"}},
	{name: "git-lfs", args: []string{"--version"}},
	{name: "go", args: []string{"version"}, re: reGo, env: []string{"GOROOT", "GOPATH", "GOPROXY", "GOSUMDB"}},
	{name: "node", args: []string{"--version"}, re: reNode, env: []string{"NODE_ENV", "NODE_OPTIONS"}},
	{name: "npm", args: []string{"--version"}, env: []string{"NPM_CONFIG_REGISTRY"}},
	{name: "pnpm", args: []string{"--version"}},
	{name: "yarn", args: []string{"--version"}},
	{name: "bun", args: []string{"--version"}},
	{name: "deno", args: []string{"--version"}},
	{name: "python", args: []string{"--version"}, env: []string{"VIRTUAL_ENV", "PYTHONPATH"}},
	{name: "python3", args: []string{"--version"}},
	{name: "pip", args: []string{"--version"}},
	{name: "uv", args: []string{"--version"}},
	{name: "java", args: []string{"-version"}, re: reJava, env: []string{"JAVA_HOME"}},
	{name: "javac", args: []string{"-version"}, re: reJava},
	{name: "mvn", args: []string{"-version"}, env: []string{"MAVEN_HOME", "M2_HOME"}},
	{name: "gradle", args: []string{"--version"}},
	{name: "kotlin", args: []string{"-version"}},
	{name: "rustc", args: []string{"--version"}, env: []string{"RUSTUP_HOME", "CARGO_HOME"}},
	{name: "cargo", args: []string{"--version"}},
	{name: "dotnet", args: []string{"--version"}, env: []string{"DOTNET_ROOT"}},
	{name: "gcc", args: []string{"--version"}},
	{name: "g++", args: []string{"--version"}},
	{name: "clang", args: []string{"--version"}},
	{name: "clang++", args: []string{"--version"}},
	{name: "make", args: []string{"--version"}},
	{name: "cmake", args: []string{"--version"}},
	{name: "ninja", args: []string{"--version"}},
	{name: "msbuild", args: []string{"-version"}},
	{name: "docker", args: []string{"--version"}, re: reDocker, env: []string{"DOCKER_HOST"}},
	{name: "podman", args: []string{"--version"}},
	{name: "kubectl", args: []string{"version", "--client"}, re: reKubectl},
	{name: "helm", args: []string{"version", "--short"}},
	{name: "terraform", args: []string{"version"}},
	{name: "aws", args: []string{"--version"}},
	{name: "az", args: []string{"version"}},
	{name: "gcloud", args: []string{"version"}},
	{name: "mysql", args: []string{"--version"}},
	{name: "psql", args: []string{"--version"}, env: []string{"PGDATA", "PGUSER"}},
	{name: "redis-cli", args: []string{"--version"}},
	{name: "mongosh", args: []string{"--version"}},
	{name: "nginx", args: []string{"-v"}},
	{name: "php", args: []string{"--version"}},
	{name: "composer", args: []string{"--version"}},
	{name: "ruby", args: []string{"--version"}, env: []string{"GEM_HOME"}},
	{name: "gem", args: []string{"--version"}},
	{name: "perl", args: []string{"--version"}},
	{name: "lua", args: []string{"-v"}},
	{name: "hugo", args: []string{"version"}},
	{name: "protoc", args: []string{"--version"}},
	{name: "vim", args: []string{"--version"}},
	{name: "nvim", args: []string{"--version"}},
	{name: "code", args: []string{"--version"}},
	{name: "wget", args: []string{"--version"}},
	{name: "curl", args: []string{"--version"}},
	{name: "ssh", args: []string{"-V"}},
	{name: "7z"},
}
