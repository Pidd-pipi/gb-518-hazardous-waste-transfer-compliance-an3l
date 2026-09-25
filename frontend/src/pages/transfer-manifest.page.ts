
import { Component } from '@angular/core'; import { EntityPageComponent } from '../components/entity-page.component'; import { ENTITY_CONFIGS } from '../types/status'; import { TransferManifestStore } from '../stores/transfer-manifest.store';
@Component({ selector: 'app-transfer-manifest-page', standalone: true, imports: [EntityPageComponent], template: `<app-entity-page [config]="config" [store]="store"/>` })
export class TransferManifestPage { readonly config = ENTITY_CONFIGS[2]; constructor(readonly store: TransferManifestStore) {} }
