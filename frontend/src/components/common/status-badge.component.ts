
import { Component, Input } from '@angular/core'; import { statusTone } from '../../utils/format';
@Component({ selector: 'app-status-badge', standalone: true, template: `<span [class]="'status status--' + tone">{{ label }}</span>` })
export class StatusBadgeComponent { @Input({ required: true }) status = ''; get tone() { return statusTone(this.status); } get label() { return this.status.replaceAll('_', ' '); } }
