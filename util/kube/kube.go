package kube

import (
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/argoproj/argo-cd/common"
)

// GetAppInstanceIdentifier returns the application instance name from annotations else labels
func GetAppInstanceIdentifier(un *unstructured.Unstructured, key string) string {
	return getAppInstanceAnnotation(un, key)
}

func getAppInstanceLabel(un *unstructured.Unstructured, key string) string {
	return un.GetLabels()[key]
}

func getAppInstanceAnnotation(un *unstructured.Unstructured, key string) string {
	return un.GetAnnotations()[key]
}

// SetAppInstanceIdentifier the recommended app.kubernetes.io/instance label against an unstructured object
// Uses the legacy labeling if environment variable is set
func SetAppInstanceIdentifier(target *unstructured.Unstructured, key, val string) error {
	setAnnotation(target, key, val)

	if key != common.LabelKeyLegacyApplicationName {
		// we no longer label the pod template sub resources in v0.11
		return nil
	}

	return setTemplateIdentifier(target, "annotations", key, val)
}

func setLabel(target *unstructured.Unstructured, key, val string) {
	labels := target.GetLabels()

	if labels == nil {
		labels = make(map[string]string)
	}

	labels[key] = val

	target.SetLabels(labels)
}

func setAnnotation(target *unstructured.Unstructured, key, val string) {
	labels := target.GetAnnotations()

	if labels == nil {
		labels = make(map[string]string)
	}

	labels[key] = val

	target.SetAnnotations(labels)
}

func setTemplateIdentifier(target *unstructured.Unstructured, parent, key, val string) error {
	gvk := schema.FromAPIVersionAndKind(target.GetAPIVersion(), target.GetKind())
	// special case for deployment and job types: make sure that derived replicaset, and pod has
	// the application label
	switch gvk.Group {
	case "apps", "extensions":
		switch gvk.Kind {
		case kube.DeploymentKind, kube.ReplicaSetKind, kube.StatefulSetKind, kube.DaemonSetKind:
			return setTemplateIdentifierValue(target, parent, key, val)
		}
	case "batch":
		switch gvk.Kind {
		case kube.JobKind:
			return setTemplateIdentifierValue(target, parent, key, val)
		}
	}
	return nil
}

func setTemplateIdentifierValue(target *unstructured.Unstructured, parent, key, val string) error {
	parentValues, ok, err := unstructured.NestedMap(target.UnstructuredContent(), "spec", "template", "metadata", parent)
	if err != nil {
		return err
	}
	if !ok || parentValues == nil {
		parentValues = make(map[string]interface{})
	}
	parentValues[key] = val
	err = unstructured.SetNestedMap(target.UnstructuredContent(), parentValues, "spec", "template", "metadata", parent)
	if err != nil {
		return err
	}

	return nil
}
