
import { Component } from '@angular/core'; import { EntityPageComponent } from '../components/entity-page.component'; import { ENTITY_CONFIGS } from '../types/status'; import { CarrierProfileStore } from '../stores/carrier-profile.store';
@Component({ selector: 'app-carrier-profile-page', standalone: true, imports: [EntityPageComponent], template: `<app-entity-page [config]="config" [store]="store"/>` })
export class CarrierProfilePage { readonly config = ENTITY_CONFIGS[1]; constructor(readonly store: CarrierProfileStore) {} }
