
import { Component } from '@angular/core'; import { EntityPageComponent } from '../components/entity-page.component'; import { ENTITY_CONFIGS } from '../types/status'; import { ComplianceCheckStore } from '../stores/compliance-check.store';
@Component({ selector: 'app-compliance-check-page', standalone: true, imports: [EntityPageComponent], template: `<app-entity-page [config]="config" [store]="store"/>` })
export class ComplianceCheckPage { readonly config = ENTITY_CONFIGS[3]; constructor(readonly store: ComplianceCheckStore) {} }
