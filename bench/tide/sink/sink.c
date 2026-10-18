/* The opaque sink the QuarkLang tide program calls: it stores the pointer in a volatile global, so
 * the compiler cannot prove the string dead, and releases the string the language's toString()
 * allocated. The compiled runtime's string builders are counted-slab allocations (stage 2, D1), so
 * the owner of the bytes is ql_free — libc free would abort with "invalid size". */
void ql_free(void* p);

void *volatile qtide_sink_keep;

void qtide_sink(char *s) {
    qtide_sink_keep = s;
}

void qtide_release(char *s) {
    ql_free(s);
}
