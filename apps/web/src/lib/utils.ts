/*
 * The shadcn registry now imports `cn` from the `cn` package directly, but
 * components.json still points its `utils` alias here. Re-export rather than
 * reimplement, so there is only ever one class-merging function in the app.
 */
export { cn } from 'cn'
