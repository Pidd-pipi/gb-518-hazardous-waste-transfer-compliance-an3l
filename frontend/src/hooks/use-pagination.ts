
import { signal } from '@angular/core';

export function createPagination(total: () => number, initialPageSize = 20) {
  const page = signal(1);
  const pageSize = signal(initialPageSize);
  const pages = () => Math.max(1, Math.ceil(total() / pageSize()));
  const reset = () => page.set(1);
  const previous = () => page.update((value) => Math.max(1, value - 1));
  const next = () => page.update((value) => Math.min(pages(), value + 1));
  return { page, pageSize, pages, reset, previous, next };
}
