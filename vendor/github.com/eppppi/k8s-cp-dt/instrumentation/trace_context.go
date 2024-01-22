package instrumentation

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/meta"

	mergelogpb "github.com/eppppi/k8s-cp-dt/mergelog/src/pkg/grpc"
)

const (
	KOC_KEY       = "eppppi.github.io/koc"
	NUM_ANC_CPIDS = 10
)

type TraceContext struct {
	Cpid     string   `json:"cpid"`
	AncCpids []string `json:"ancCpids"`
}

// DeepCopyTraceContext deep copies a trace context
func (tctx *TraceContext) DeepCopyTraceContext() *TraceContext {
	if tctx == nil {
		return nil
	}
	return newTraceContext(tctx.Cpid, tctx.AncCpids)
}

// ValidateTctx validates if tctx is valid
func (tctx *TraceContext) validateTctx() error {
	if tctx == nil {
		return fmt.Errorf("validation failed: tctx is nil")
	}
	if tctx.Cpid == "" {
		return fmt.Errorf("validation failed: cpid is empty string")
	}
	if len(tctx.AncCpids) > NUM_ANC_CPIDS {
		return fmt.Errorf("validation failed: ancCpids (limit: %d) is too long %d", NUM_ANC_CPIDS, len(tctx.AncCpids))
	}
	if containsString(tctx.AncCpids, tctx.Cpid) {
		return fmt.Errorf("validation failed: cpid is included in ancCpids")
	}
	return nil
}

type KeyWithTraceContexts struct {
	Key       interface{}
	TraceCtxs []*TraceContext
}

// NewRootTraceContext creates a new root trace context and send mergelog of it
func NewRootTraceContextAndSendMergelog(message, by string) *TraceContext {
	cpid, _ := generateCpid()
	newTctx := newTraceContext(cpid, []string{})

	sendMergelog(cpid, []string{}, mergelogpb.CauseType_CAUSE_TYPE_NEW_CHANGE, message, by)

	return newTctx
}

// newTraceContext creates a new trace context (deep copy)
func newTraceContext(cpid string, ancCpids []string) *TraceContext {
	newAncCpids := make([]string, len(ancCpids))
	copy(newAncCpids, ancCpids)
	return &TraceContext{
		Cpid:     cpid,
		AncCpids: newAncCpids,
	}
}

// GetCpid gets cpid
func (tctx *TraceContext) GetCpid() string {
	if tctx == nil {
		return ""
	}
	return tctx.Cpid
}

// SetCpid sets cpid
func (tctx *TraceContext) SetCpid(cpid string) {
	if tctx == nil {
		return
	}
	tctx.Cpid = cpid
}

// GetAncCpids gets ancestor cpids
func (tctx *TraceContext) GetAncCpids() []string {
	if tctx == nil {
		return nil
	}
	return tctx.AncCpids
}

// SetAncCpids sets ancestor cpids
func (tctx *TraceContext) SetAncCpids(ancCpids []string) {
	if tctx == nil {
		return
	}
	tctx.AncCpids = ancCpids
}

// GenerateCpid generates a cpid
func generateCpid() (string, error) {
	newUuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return newUuid.String(), nil
}

// REFACTOR: from interface{} to runtime.Object
// GetTraceContext returns trace context (maybe nil)
func GetTraceContext(objInterface interface{}) *TraceContext {
	obj, err := meta.Accessor(objInterface)
	if err != nil {
		return nil
	}

	annotations := obj.GetAnnotations()
	if ctxs, ok := annotations[KOC_KEY]; ok {
		var traceCtxs TraceContext
		err := json.Unmarshal([]byte(ctxs), &traceCtxs)
		if err != nil {
			log.Printf("warning: unmarshal error: %v\n, returning nil", err)
			return nil
		}
		return &traceCtxs
	} else {
		log.Println("no trace context in this object, returning nil")
		return nil
	}
}

// REFACTOR: from interface{} to runtime.Object
// SetTraceContext sets trace context
func SetTraceContext(objInterface interface{}, traceCtx *TraceContext) error {
	if err := traceCtx.validateTctx(); err != nil {
		return err
	}

	obj, err := meta.Accessor(objInterface)
	if err != nil {
		return err
	}
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	ctxsString, err := json.Marshal(traceCtx)
	if err != nil {
		log.Printf("warning: marshal error: %v", err)
	}
	annotations[KOC_KEY] = string(ctxsString)
	obj.SetAnnotations(annotations)

	return nil
}

// ====================================

// mergeTctxs merges tctxs and returns the new tctx (maybe nil), dest cpid, and source cpids
// if sourceCpids is nil, no need to send a mergelog.
func mergeTctxs(tctxs []*TraceContext) (retTctx *TraceContext, destCpid string, sourceCpids []string) {
	// validate args
	if len(tctxs) == 0 {
		return nil, "", nil
	}
	for _, tctx := range tctxs {
		if err := tctx.validateTctx(); err != nil {
			log.Printf("tctx(%v) is invalid, so cannot add: %v\n", tctx, err)
			return nil, "", nil
		}
	}

	cg := createCpidGraph(tctxs)
	if len(cg.roots) == 0 { // TODO: This should not happen(?), and this func should not return nil
		return nil, "", nil
	}
	if len(cg.roots) == 1 {
		log.Println("only 1 root exists, no need to merge")
		for k := range cg.roots {
			destCpid = k
		}
		sourceCpids = nil
	} else {
		destCpid, _ = generateCpid()
		sourceCpids = make([]string, len(cg.roots))
		{
			i := 0
			for k := range cg.roots {
				sourceCpids[i] = k
				i++
			}
		}
		cg.addTraceContext(&TraceContext{destCpid, sourceCpids})
	}

	min := NUM_ANC_CPIDS
	if len(cg.roots[destCpid]) < min {
		min = len(cg.roots[destCpid])
	}
	newAncCpids := cg.roots[destCpid][:min]
	retTctx = newTraceContext(destCpid, newAncCpids)

	return retTctx, destCpid, sourceCpids
}

// use case: finding best ancCpids in each controller, *not for trace server*
type cpidGraph struct {
	// 深さ１に固定する。祖先の祖先が出てきた場合はrootに直接繋ぐようにする。
	// NOTE: any root key MUST NOT be included in as a value of other keys.
	roots map[string][]string
}

func newEmptyCpidGraph() *cpidGraph {
	return &cpidGraph{
		roots: make(map[string][]string),
	}
}

// CreateCpidGraph creates a cpid graph from trace contexts
// all trace contexts must be valid
func createCpidGraph(tctxs []*TraceContext) *cpidGraph {
	cg := newEmptyCpidGraph()
	for _, tctx := range tctxs {
		cg.addTraceContext(tctx)
	}
	return cg
}

// fronter is newer
func (cg *cpidGraph) addTraceContext(tctx *TraceContext) {
	if err := tctx.validateTctx(); err != nil {
		log.Printf("tctx(%v) is invalid, so cannot add: %v\n", tctx, err)
		return
	}
	// step 1: check if any of tctx.AncCpids matches in any roots.
	// if so, add values of the root to cg.roots[tctx.Cpid] and delete the root from cg.roots. if not, do nothing.
	tmpAncCpidsOfTctx := make([]string, len(tctx.AncCpids))
	copy(tmpAncCpidsOfTctx, tctx.AncCpids)
	for k := range cg.roots {
		if containsString(tctx.AncCpids, k) { // 自分の先祖(k)を発見した場合、そのさらに先祖(cg.roots[k])を自分のtmpAncCpidsOfTctxに追加していく
			for _, v := range cg.roots[k] {
				if !containsString(tmpAncCpidsOfTctx, v) {
					tmpAncCpidsOfTctx = append(tmpAncCpidsOfTctx, v)
				}
			}
			delete(cg.roots, k)
		}
	}

	// step 2: check if tctx.Cpid is included in any values or roots.
	// if so, add tctx.AncCpids to the values of the root. if not, add tctx to roots.
	cpidIsIncluded := false
	for k := range cg.roots {
		if k == tctx.Cpid || containsString(cg.roots[k], tctx.Cpid) {
			for i := len(tmpAncCpidsOfTctx) - 1; i >= 0; i-- {
				if !containsString(cg.roots[k], tmpAncCpidsOfTctx[i]) {
					cg.roots[k] = append([]string{tmpAncCpidsOfTctx[i]}, cg.roots[k]...)
				}
			}
			cpidIsIncluded = true
		}
	}
	if !cpidIsIncluded {
		cg.roots[tctx.Cpid] = tmpAncCpidsOfTctx
	}
}

// ======= utils =======

// Contains returns true if str is included in slice
// implemented because slices.Contains is introduced in go 1.21 (which is not supported in k8s 1.27)
func containsString(slice []string, str string) bool {
	if slice == nil {
		return false
	}
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
