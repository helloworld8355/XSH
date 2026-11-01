package secscan

import (
	"fmt"

	"xsh/internal/audit"
	"xsh/internal/license"
)

// Guard 授权门禁：license 验证 + 显式声明 + 审计记录。
type Guard struct {
	Payload *license.Payload
}

// NewGuard 校验授权，通过后写审计日志。
func NewGuard(licensePath, authorizedBy string, declared bool) (*Guard, error) {
	if !declared {
		return nil, fmt.Errorf("安全评估功能需显式声明 --authorized，确认你已获得目标系统的书面授权")
	}
	if licensePath == "" {
		return nil, fmt.Errorf("缺少 --license 授权文件（评估授权的申请与使用说明见 README）")
	}
	p, err := license.LoadVerify(licensePath)
	if err != nil {
		return nil, err
	}
	_ = audit.Log(audit.Entry{Cmd: "secscan.grant", Result: "granted", Licensee: p.Licensee, Args: authorizedBy})
	return &Guard{Payload: p}, nil
}

// Audit 记录一次扫描动作。
func (g *Guard) Audit(action, target, result string) {
	_ = audit.Log(audit.Entry{Cmd: action, Target: target, Result: result, Licensee: g.Payload.Licensee})
}
