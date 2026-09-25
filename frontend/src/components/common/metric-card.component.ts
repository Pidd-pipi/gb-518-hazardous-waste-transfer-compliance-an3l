
import { Component, Input } from '@angular/core';
@Component({ selector: 'app-metric-card', standalone: true, template: `<section class="metric"><span>{{ label }}</span><strong>{{ value }}</strong><small>{{ detail }}</small></section>` })
export class MetricCardComponent { @Input() label = ''; @Input() value: string | number = 0; @Input() detail = ''; }
