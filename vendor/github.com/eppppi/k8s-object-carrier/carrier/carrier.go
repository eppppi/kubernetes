package carrier

import (
	"encoding/json"
	"log"
	// "strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// const KOC_PREFIX = "eppppi/koc/"
const KOC_KEY = "eppppi.github.io/koc"

// func isTraceKey(key string) bool {
// 	return strings.HasPrefix(key, KOC_PREFIX)
// }

// OpenTelemetry carrier using annotation of kubernetes object
type K8sObjAntCarrier struct {
	metav1.Object
}

func NewK8sAntCarrierFromInterface(objInterface interface{}) (*K8sObjAntCarrier, error) {
	obj, err := meta.Accessor(objInterface)
	if err != nil {
		return nil, err
	}
	return NewK8sAntCarrierFromObj(obj)
}

func NewK8sAntCarrierFromObj(obj metav1.Object) (*K8sObjAntCarrier, error) {
	return &K8sObjAntCarrier{obj}, nil
}

func (objCarrier *K8sObjAntCarrier) Get(key string) string {
	annotations := objCarrier.GetAnnotations()
	if ctxs, ok := annotations[KOC_KEY]; ok {
		var mapCtxs map[string]string
		err := json.Unmarshal([]byte(ctxs), &mapCtxs)
		if err != nil {
			log.Printf("warning: unmarshal error: %v", err)
			return ""
		}
		if ctx, ok := mapCtxs[key]; ok {
			return ctx
		} else {
			log.Printf("warning: key %s is not found", key)
			return ""
		}
	} else {
		log.Println("no trace context in this object")
		return ""
	}
}

func (objCarrier *K8sObjAntCarrier) Set(key string, value string) {
	annotations := objCarrier.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	var mapCtxs map[string]string
	if ctxs, ok := annotations[KOC_KEY]; ok {
		// ctxsが存在する時、既存のctxsを取得してkey-valueを追加する
		err := json.Unmarshal([]byte(ctxs), &mapCtxs)
		if err != nil {
			log.Printf("warning: unmarshal error: %v", err)
			return
		}
	} else {
		// ctxsが存在しない時、新しいctxsを作成してkey-valueを追加する
		mapCtxs = make(map[string]string)
	}
	mapCtxs[key] = value
	ctxsString, err := json.Marshal(mapCtxs)
	if err != nil {
		log.Printf("warning: marshal error: %v", err)
	}
	annotations[KOC_KEY] = string(ctxsString)
	objCarrier.SetAnnotations(annotations)
}

func (objCarrier *K8sObjAntCarrier) Keys() []string {
	annotations := objCarrier.GetAnnotations()
	if ctxs, ok := annotations[KOC_KEY]; ok {
		var mapCtxs map[string]string
		err := json.Unmarshal([]byte(ctxs), &mapCtxs)
		if err != nil {
			log.Printf("warning: unmarshal error: %v", err)
			return []string{}
		}
		keys := make([]string, 0, len(mapCtxs))
		for k := range mapCtxs {
			keys = append(keys, k)
		}
		return keys
	} else {
		return []string{}
	}
}
