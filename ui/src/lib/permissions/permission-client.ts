import {
  ACTION_BITS,
  DataScope,
  FieldAccess,
  type ActionName,
  type PermissionManifest,
  type Resource,
  type ResourceDetail,
} from "@/types/permission";
import { BloomFilter } from "./bloom-filter";

/**
 * High-performance permission client for sub-millisecond permission checks
 *
 * Uses:
 * - Bloom filters for fast negative checks
 * - Bitfield operations for standard permissions
 * - Local caching for instant lookups
 *
 * @example
 * const client = new PermissionClient(manifest);
 * if (client.can('shipment', 'create')) {
 *   // User can create shipments
 * }
 */
export class PermissionClient {
  private manifest: PermissionManifest;
  private bloom: BloomFilter | null = null;

  constructor(manifest: PermissionManifest) {
    this.manifest = manifest;
    // Bloom filter would be initialized if provided in manifest
    // For now, we'll skip bloom filter for simplicity
  }

  /**
   * Check if user has permission for a specific action on a resource
   * Sub-millisecond performance using bitfield operations
   *
   * @param resource The resource type (e.g., 'shipment', 'user')
   * @param action The action to perform (e.g., 'create', 'read')
   * @returns true if user has permission, false otherwise
   */
  can(resource: Resource, action: ActionName): boolean {
    // Quick negative check using bloom filter (if available)
    if (this.bloom && !this.bloom.test(`${resource}:${action}`)) {
      return false;
    }

    const perms = this.manifest.resources[resource];

    // No permissions for this resource
    if (!perms) {
      return false;
    }

    // Simple bitfield permission (number)
    if (typeof perms === "number") {
      const actionBit = ACTION_BITS[action];
      if (!actionBit) {
        return false; // Unknown action
      }
      return (perms & actionBit) > 0;
    }

    // Complex permission (ResourceDetail)
    return this.evaluateComplex(perms, action);
  }

  /**
   * Evaluate complex resource permissions
   */
  private evaluateComplex(perms: ResourceDetail, action: ActionName): boolean {
    // Check standard operations
    const actionBit = ACTION_BITS[action];
    if (actionBit && (perms.standardOps & actionBit) > 0) {
      return true;
    }

    // Check extended operations
    if (perms.extendedOps && perms.extendedOps.includes(action)) {
      return true;
    }

    return false;
  }

  /**
   * Check if user can perform any of the specified actions
   *
   * @param resource The resource type
   * @param actions Array of actions to check
   * @returns true if user has at least one permission
   */
  canAny(resource: Resource, actions: ActionName[]): boolean {
    return actions.some((action) => this.can(resource, action));
  }

  /**
   * Check if user can perform all of the specified actions
   *
   * @param resource The resource type
   * @param actions Array of actions to check
   * @returns true if user has all permissions
   */
  canAll(resource: Resource, actions: ActionName[]): boolean {
    return actions.every((action) => this.can(resource, action));
  }

  /**
   * Get data scope for a resource
   *
   * @param resource The resource type
   * @returns The data scope (all, organization, own, none)
   */
  getDataScope(resource: Resource): DataScope | null {
    const perms = this.manifest.resources[resource];

    if (!perms) {
      return null;
    }

    if (typeof perms === "number") {
      return DataScope.All; // Simple permissions default to "all"
    }

    return perms.dataScope;
  }

  /**
   * Get field access for a resource
   *
   * @param resource The resource type
   * @param field The field name
   * @returns The field access level
   */
  getFieldAccess(resource: Resource, field: string): FieldAccess | null {
    const perms = this.manifest.resources[resource];

    if (!perms || typeof perms === "number") {
      return null; // No field rules
    }

    const fieldRules = perms.fieldRules;
    if (!fieldRules) {
      return null;
    }

    // Check denied fields
    if (fieldRules.denied.includes(field)) {
      return FieldAccess.Hidden;
    }

    // Check read-only fields
    if (fieldRules.readOnly.includes(field)) {
      return FieldAccess.ReadOnly;
    }

    // Check allowed fields
    if (fieldRules.allowed.includes(field)) {
      return FieldAccess.ReadWrite;
    }

    // Check masked fields
    if (fieldRules.masked.includes(field)) {
      return FieldAccess.ReadOnly; // Masked is effectively read-only
    }

    // Default: no access
    return FieldAccess.Hidden;
  }

  /**
   * Check if a field is accessible (read or write)
   *
   * @param resource The resource type
   * @param field The field name
   * @returns true if field is accessible
   */
  canAccessField(resource: Resource, field: string): boolean {
    const access = this.getFieldAccess(resource, field);
    return access !== null && access !== "hidden";
  }

  /**
   * Check if a field is writable
   *
   * @param resource The resource type
   * @param field The field name
   * @returns true if field is writable
   */
  canWriteField(resource: Resource, field: string): boolean {
    const access = this.getFieldAccess(resource, field);
    return access === "read_write" || access === "write_only";
  }

  /**
   * Get current user ID
   */
  getUserId(): string {
    return this.manifest.userId;
  }

  /**
   * Get current organization ID
   */
  getCurrentOrganization(): string {
    return this.manifest.currentOrg;
  }

  /**
   * Get available organization IDs
   */
  getAvailableOrganizations(): string[] {
    return this.manifest.availableOrgs;
  }

  /**
   * Get all resources with permissions
   */
  getResources(): Resource[] {
    return Object.keys(this.manifest.resources);
  }

  /**
   * Check if permissions have expired
   */
  isExpired(): boolean {
    return Date.now() / 1000 > this.manifest.expiresAt;
  }

  /**
   * Get time until expiration (in seconds)
   */
  getTimeUntilExpiration(): number {
    return Math.max(0, this.manifest.expiresAt - Date.now() / 1000);
  }

  /**
   * Update the manifest (e.g., after organization switch)
   */
  updateManifest(manifest: PermissionManifest): void {
    this.manifest = manifest;
    this.bloom = null; // Reset bloom filter
  }

  /**
   * Get the full manifest (for debugging)
   */
  getManifest(): PermissionManifest {
    return this.manifest;
  }
}
