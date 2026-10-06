# The standard library

**Minimal core, everything else as an extension.** The language core provides *pointers and basic
types* only. There are no built-in containers: every collection, smart pointer and higher-level
facility lives here and must be imported.

```quark
import "vec";                 // the collection library lives here, not in the language
Vec<int> v = [1, 2, 3];       // [...] is an overloadable notation, so the binding is the library's
```

Rules that follow from the principle:

1. **A new container is a new library file**, never a new built-in type. The core does not know the
   name `Vec`, and it must not learn it: `[...]` binds through the literal protocol
   (`__literal__()` + `__element__`) that any type can implement, and `for x : c` iterates through
   `size()` + `get(int)`.
2. **The core may shrink, never grow.** The two built-in containers still present (`List`, `HashTable`)
   are legacy: they are being migrated onto library types (`Vec` for `List`), and the count is frozen
   by a test so it can only go down.
3. **Smart pointers belong here too.** The core keeps raw pointers/references as primitives; permission
   carrying references (BioLang's typed smart references) are a library-level extension on top of them.
