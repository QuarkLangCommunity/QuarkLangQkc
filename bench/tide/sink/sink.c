/* The opaque sink the QuarkLang tide program calls: it stores the pointer in a volatile global, so
 * the compiler cannot prove the string dead, and releases the string the language's toString()
 * allocated. Built as libtide.a so qkc can link it through `library tide { ... }`. */
#include <stdlib.h>

void *volatile qtide_sink_keep;

void qtide_sink(char *s) {
    qtide_sink_keep = s;
}

void qtide_release(char *s) {
    free(s);
}
