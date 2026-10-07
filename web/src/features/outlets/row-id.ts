// outletRowId is the id of an outlet's row header, so the page can move focus
// to an outlet it has just added.
export function outletRowId(id: number): string {
  return `outlet-${id}`;
}
