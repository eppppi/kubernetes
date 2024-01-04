package instrumentation

import (
	"encoding/json"
	// "fmt"
	"log"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	KOC_KEY = "eppppi.github.io/koc"
)

type TraceContext struct {
	Cpid     string   `json:"cpid"`
	AncCpids []string `json:"ancCpids"`
}

// GetCpid gets cpid
func (tc *TraceContext) GetCpid() string {
	return tc.Cpid
}

// SetCpid sets cpid
func (tc *TraceContext) SetCpid(cpid string) {
	tc.Cpid = cpid
}

// GetAncCpids gets ancestor cpids
func (tc *TraceContext) GetAncCpids() []string {
	return tc.AncCpids
}

// SetAncCpids sets ancestor cpids
func (tc *TraceContext) SetAncCpids(ancCpids []string) {
	tc.AncCpids = ancCpids
}

// GenerateCpid generates a cpid
func GenerateCpid() (string, error) {
	newUuid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return newUuid.String(), nil
}

// func GetCpid(objInterface interface{}) string {
// 	obj, err := meta.Accessor(objInterface)
// 	if err != nil {
// 		return ""
// 	}

// 	annotations := obj.GetAnnotations()
// 	if ctxs, ok := annotations[KOC_KEY]; ok {
// 		var traceCtxs TraceContext
// 		err := json.Unmarshal([]byte(ctxs), &traceCtxs)
// 		if err != nil {
// 			log.Printf("warning: unmarshal error: %v\n", err)
// 			return ""
// 		}
// 		return traceCtxs.Cpid
// 	} else {
// 		log.Println("no trace context in this object")
// 		return ""
// 	}
// }

// func SetCpid(objInterface interface{}, cpid string) error {
// 	obj, err := meta.Accessor(objInterface)
// 	if err != nil {
// 		return err
// 	}
// 	annotations := obj.GetAnnotations()
// 	if annotations == nil {
// 		annotations = make(map[string]string)
// 	}
// 	var traceCtxs TraceContext
// 	if ctxs, ok := annotations[KOC_KEY]; ok {
// 		// traceCtxsが存在する時、既存のctxsを取得してkey-valueを追加する
// 		err := json.Unmarshal([]byte(ctxs), &traceCtxs)
// 		if err != nil {
// 			log.Printf("warning: unmarshal error: %v", err)
// 			return err
// 		}
// 	} else {
// 		// traceCtxsが存在しない時、新しいctxsを作成してkey-valueを追加する
// 		traceCtxs = TraceContext{}
// 	}
// 	traceCtxs.Cpid = cpid
// 	ctxsString, err := json.Marshal(traceCtxs)
// 	if err != nil {
// 		log.Printf("warning: marshal error: %v", err)
// 	}
// 	annotations[KOC_KEY] = string(ctxsString)
// 	obj.SetAnnotations(annotations)

// 	return nil
// }

// // GetAncCpids returns ancestor cpids
// func GetAncCpids(objInterface interface{}) []string {
// 	obj, err := meta.Accessor(objInterface)
// 	if err != nil {
// 		return nil
// 	}

// 	annotations := obj.GetAnnotations()
// 	if ctxs, ok := annotations[KOC_KEY]; ok {
// 		var traceCtxs TraceContext
// 		err := json.Unmarshal([]byte(ctxs), &traceCtxs)
// 		if err != nil {
// 			log.Printf("warning: unmarshal error: %v\n", err)
// 			return nil
// 		}
// 		return traceCtxs.AncCpids
// 	} else {
// 		log.Println("no trace context in this object")
// 		return nil
// 	}
// }

// GetTraceContext returns trace context
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
			log.Printf("warning: unmarshal error: %v\n", err)
			return nil
		}
		return &traceCtxs
	} else {
		log.Println("no trace context in this object, creating new trace context")
		return &TraceContext{}
	}
}

// SetTraceContext sets trace context
func SetTraceContext(objInterface interface{}, traceCtx *TraceContext) error {
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
