
export function formatDate(value: string): string {
  return value ? new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) : '-';
}
export function statusTone(status: string): 'success' | 'warning' | 'danger' | 'neutral' {
	if (/approved|accepted|released|completed|signed|closed|pass|ready|online|cleared|succeeded|verified|received|active/.test(status)) return 'success';
	if (/failed|fail|rejected|critical|scrap|discard|revoked|urgent|expired|suspended/.test(status)) return 'danger';
	if (/hold|warning|review|pending|restricted|limited|quarantine|submitted|in_transit|escalated/.test(status)) return 'warning';
	return 'neutral';
}

export function daysUntil(value?: string): number | null {
	if (!value) return null;
	return Math.ceil((new Date(value).getTime() - Date.now()) / 86_400_000);
}
