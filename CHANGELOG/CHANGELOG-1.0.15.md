<a name="v1.0.15"></a>

## [v1.0.15] - 2026-11-24

### Bug Fixes

- **sort_map:** make nested `SortMap`/`MapList` struct fields round-trip correctly with
  yaml.v3 and encoding/json (c7ed018)
    - Switch `MarshalJSON`/`MarshalYAML` to value receivers: the yaml.v3 encoder never takes the
      address of non-addressable value-typed struct fields, so the previous pointer-receiver
      marshalers were silently skipped on such fields and produced an empty `{}` or a reflected
      internal-layout mapping instead of the ordered key/value output
    - Rework `UnmarshalJSON`/`UnmarshalYAML`: zero-value containers are lazily initialized on the
      first write or decode (race-safe under concurrent first writes), null/empty nodes reset the
      container to empty, and decoding into an already-initialized field fully replaces its
      content
    - Reject negative indexes when decoding a MapList from JSON/YAML, with the offending index
      included in the error message

### Code Refactoring

- **sort_map:** rebuild the container API around exported concrete types only (c7ed018)
    - `SortMap[K, V]`/`MapList[V]` become exported structs; `MapList[V]` embeds `SortMap[int, V]`
      by value and the duplicated `Get`/`Contains`/`Len`/`Clear`/`Range`/`Indexes` methods and the
      marshal wrappers are dropped in favor of method promotion, which keeps the value-receiver
      marshalers reachable on value-typed struct fields
    - Add `yaml_nested_test.go` covering nested-field decoding, null/empty-node semantics, content
      replacement and JSON/YAML round-trip consistency, and adjust the existing unit tests for the
      new types
    - Sync the zh-CN library guide `docs/guide/zh-CN/libs/sort_map.md` with the new API surface and
      the yaml.v3 interface-field limitation

### BREAKING CHANGE

sort_map API surface: the behavior interfaces are removed and the exported names now denote
concrete structs (c7ed018):

1. `SortMap`/`MapList` change from interfaces to exported concrete structs; the v1.0.14
   implementations were unexported (`sortMap`/`mapList`) and are now the public types themselves.
2. `Map()`/`List()` return concrete value types instead of interface values. The zero value of
   `SortMap`/`MapList` is an empty, ready-to-use container: pre-initialization for JSON decoding is
   no longer required, and nested YAML decoding works on plain value-typed struct fields.
3. `mock_sort_map.go` (`MockSortMap`/`MockMapList`/`MockMapIndexes`) is deleted from the package;
   no gomock injection points are provided anymore. Code that needs mockability should define the
   interface in its own layer, or prefer testing against the concrete containers directly.
4. Business struct fields that require JSON/YAML deserialization must be declared as the concrete
   types (or pointers to them); interface-typed fields — including user-defined wrapper interfaces —
   still cannot be deserialized by yaml.v3 (its `prepare()` never dispatches `UnmarshalYAML`
   through an interface field).

The code migration example is as follows:

Before (v1.0.14):

```go
type Config struct {
Backends sort_map.SortMap[string, string] // interface type
Slots    sort_map.MapList[string] // interface type
}

cfg := Config{
Backends: sort_map.Map[string, string](), // pre-initialization required for JSON decoding
Slots:    sort_map.List[string](),
}
if cfg.Backends == nil { // interface nil check
// ...
}
```

After (v1.0.15):

```go
type Config struct {
Backends sort_map.SortMap[string, string] // concrete struct, zero value ready to use
Slots    *sort_map.MapList[string] // pointer when a YAML null must be observable
}

var cfg Config // no pre-initialization needed
if err := yaml.Unmarshal(data, &cfg); err != nil {
// ...
}
```

Impact boundary: field/variable declarations that simply spelled `sort_map.SortMap[...]` or
`sort_map.MapList[...]` and called the container methods compile unchanged on v1.0.15 and directly
benefit from the serialization fixes. Adjustment is only required for code that assigned `nil` (or
custom implementations) to these typed variables, asserted or type-switched the result of
`Map()`/`List()`, or referenced the deleted mocks. Note that chaining pointer-receiver write
methods on a non-addressable constructor result (e.g. `sort_map.Map[string, int]().Set(...)`) was
already illegal under the v1.0.14 interface API, so it needs no migration.

[v1.0.15]: https://github.com/ClessLi/component-base/compare/v1.0.14...v1.0.15
