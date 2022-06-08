package js

import (
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja/unistring"
	"go.k6.io/k6/js/modules"
)

func wrapGoModule(mod interface{}) goja.ModuleRecord {
	k6m, ok := mod.(modules.Module)
	if !ok {
		return &wrappedBasicGoModule{m: mod}
	}
	return wrappedGoModule{m: k6m}
}

// This goja.ModuleRecord wrapper for go/js module which does not conform to modules.Module interface
type wrappedBasicGoModule struct {
	m             interface{}
	once          sync.Once
	exportedNames []string
}

func (w *wrappedBasicGoModule) Link() error { return nil }
func (w *wrappedBasicGoModule) Evaluate(rt *goja.Runtime) (goja.ModuleInstance, error) {
	o := rt.ToValue(w.m).ToObject(rt) // TODO checks
	w.once.Do(func() { w.exportedNames = o.Keys() })
	return &wrappedBasicGoModuleInstance{
		v: o,
	}, nil
}

func (w *wrappedBasicGoModule) ResolveExport(
	exportName string, set ...goja.ResolveSetElement,
) (*goja.ResolvedBinding, bool) {
	return &goja.ResolvedBinding{
		Module:      w,
		BindingName: exportName,
	}, false
}

func (w *wrappedBasicGoModule) GetExportedNames(set ...goja.ModuleRecord) []string {
	w.once.Do(func() { panic("this shouldn't happen") })
	return w.exportedNames
}

type wrappedBasicGoModuleInstance struct {
	v *goja.Object
}

func (wmi *wrappedBasicGoModuleInstance) GetBindingValue(n unistring.String) goja.Value {
	if n == "default" {
		return wmi.v
	}
	return wmi.v.Get(n.String())
}

// This goja.ModuleRecord wrapper for go/js module which conforms to modules.Module interface
type wrappedGoModule struct {
	m modules.Module
}

func (w wrappedGoModule) Link() error {
	return nil // TDOF fix
}

func (w wrappedGoModule) Evaluate(rt *goja.Runtime) (goja.ModuleInstance, error) {
	vu := rt.GlobalObject().Get("vugetter").Export().(vugetter).get() //nolint:forcetypeassert
	mi := w.m.NewModuleInstance(vu)
	return &wrappedGoModuleInstance{mi: mi, rt: rt}, nil // TODO fix
}

func (w wrappedGoModule) GetExportedNames(set ...goja.ModuleRecord) []string {
	return []string{}
}

func (w wrappedGoModule) ResolveExport(exportName string, set ...goja.ResolveSetElement) (*goja.ResolvedBinding, bool) {
	return &goja.ResolvedBinding{
		Module:      w,
		BindingName: exportName,
	}, false
}

type wrappedGoModuleInstance struct {
	mi modules.Instance
	rt *goja.Runtime
}

func (wmi *wrappedGoModuleInstance) GetBindingValue(name unistring.String) goja.Value {
	n := name.String()
	exports := wmi.mi.Exports()
	if n == "default" {
		if exports.Default == nil {
			return wmi.rt.ToValue(exports.Named)
		}
		return wmi.rt.ToValue(exports.Default)
	}
	return wmi.rt.ToValue(exports.Named[n])
}
