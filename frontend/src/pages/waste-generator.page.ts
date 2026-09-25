
import { Component } from '@angular/core'; import { EntityPageComponent } from '../components/entity-page.component'; import { ENTITY_CONFIGS } from '../types/status'; import { WasteGeneratorStore } from '../stores/waste-generator.store';
@Component({ selector: 'app-waste-generator-page', standalone: true, imports: [EntityPageComponent], template: `<app-entity-page [config]="config" [store]="store"/>` })
export class WasteGeneratorPage { readonly config = ENTITY_CONFIGS[0]; constructor(readonly store: WasteGeneratorStore) {} }
