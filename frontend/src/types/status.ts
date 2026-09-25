import type { EntityConfig } from './domain';

export type ManifestState = 'draft' | 'submitted' | 'in_transit' | 'received' | 'rejected';
export const ALL_MANIFEST_STATE: readonly ManifestState[] = ['draft', 'submitted', 'in_transit', 'received', 'rejected'];
export type CheckState = 'pending' | 'pass' | 'fail' | 'escalated';
export const ALL_CHECK_STATE: readonly CheckState[] = ['pending', 'pass', 'fail', 'escalated'];

export const ENTITY_CONFIGS: readonly EntityConfig[] = [
  { key: 'wasteGenerator', path: 'generators', label: '产废单位', statuses: ['active', 'restricted', 'suspended', 'expired'] as const, transitionRole: 'reviewer' },
  { key: 'carrierProfile', path: 'carriers', label: '承运资质', statuses: ['pending', 'verified', 'restricted', 'expired'] as const, transitionRole: 'reviewer' },
  { key: 'transferManifest', path: 'manifests', label: '转运清单', statuses: ['draft', 'submitted', 'in_transit', 'received', 'rejected'] as const, transitionRole: 'operator' },
  { key: 'complianceCheck', path: 'checks', label: '合规核验', statuses: ['pending', 'pass', 'fail', 'escalated'] as const, transitionRole: 'reviewer' }
];

export const TRANSITIONS: Readonly<Record<string, Readonly<Record<string, readonly string[]>>>> = {
  wasteGenerator: { active: ['restricted', 'suspended', 'expired'], restricted: ['active', 'suspended', 'expired'], suspended: ['active', 'expired'], expired: [] },
  carrierProfile: { pending: ['verified', 'restricted', 'expired'], verified: ['restricted', 'expired'], restricted: ['pending', 'expired'], expired: [] },
  transferManifest: { draft: ['submitted'], submitted: ['in_transit', 'rejected'], in_transit: ['received', 'rejected'], received: [], rejected: [] },
  complianceCheck: { pending: ['pass', 'fail', 'escalated'], pass: [], fail: ['escalated'], escalated: [] }
};
