package carrier

import (
	"log"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const KOC_PREFIX = "github.com/eppppi/koc/"

func isTraceKey(key string) bool {
	return strings.HasPrefix(key, KOC_PREFIX)
}

// OpenTelemetry carrier using annotation of kubernetes object
type K8sObjAntCarrier metav1.ObjectMeta

func (objMeta *K8sObjAntCarrier) Get(key string) string {
	annotations := (*metav1.ObjectMeta)(objMeta).GetAnnotations()
	if val, ok := annotations[key]; ok {
		return val
	} else {
		log.Printf("warning: key %s is not found", key)
		return ""
	}
}

func (objMeta *K8sObjAntCarrier) Set(key string, value string) {
	if !isTraceKey(key) {
		log.Printf("warning: key %s is invalid for trace key", key)
	}
	(*metav1.ObjectMeta)(objMeta).SetAnnotations(map[string]string{key: value})
}

func (objMeta *K8sObjAntCarrier) Keys() []string {
	annotations := (*metav1.ObjectMeta)(objMeta).GetAnnotations()
	keys := make([]string, 0, len(annotations))
	for k := range annotations {
		if isTraceKey(k) {
			keys = append(keys, k)
		}
	}
	return keys
}
