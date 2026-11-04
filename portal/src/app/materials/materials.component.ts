/**
 * IANUS Simulation - Materials Page Component
 *
 * Two sections on one page:
 *  - StrömungsRaum materials: the singleton MaterialCatalog's browse view
 *    (Refresh -> status.availableMaterials). No "+" - there's only one.
 *  - Submissions: HttpBin-style grid of MaterialSubmission cards. "+" opens
 *    a dialog listing the account's SKZ Materials (a different provider's
 *    CRD); picking one and confirming embeds its full spec into a new
 *    MaterialSubmission.
 */
import { Component, CUSTOM_ELEMENTS_SCHEMA, computed, inject, signal } from '@angular/core';
import * as LuigiClient from '@luigi-project/client';
import {
  ButtonComponent,
  DialogComponent,
  DynamicPageComponent,
  DynamicPageTitleComponent,
  IconComponent,
  InputComponent,
  LabelComponent,
  OptionComponent,
  SelectComponent,
  TextComponent,
  TitleComponent,
} from '@ui5/webcomponents-ngx';

import '@ui5/webcomponents-icons/dist/add.js';
import '@ui5/webcomponents-icons/dist/decline.js';
import '@ui5/webcomponents-icons/dist/refresh.js';
import '@ui5/webcomponents-icons/dist/accept.js';
import '@ui5/webcomponents-icons/dist/action.js';
import '@ui5/webcomponents-icons/dist/navigation-left-arrow.js';
import '@ui5/webcomponents-icons/dist/navigation-right-arrow.js';

import { MaterialCatalog, MaterialSubmission, MaterialsService, SkzMaterial } from './materials.service';

@Component({
  selector: 'app-materials',
  standalone: true,
  imports: [
    DynamicPageComponent,
    DynamicPageTitleComponent,
    TitleComponent,
    LabelComponent,
    TextComponent,
    ButtonComponent,
    DialogComponent,
    SelectComponent,
    OptionComponent,
    IconComponent,
    InputComponent,
  ],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './materials.component.html',
  styleUrl: './materials.component.scss',
})
export class MaterialsComponent {
  private client = inject(MaterialsService);

  private static readonly CATALOG_PAGE_SIZE = 5;

  public catalog = signal<MaterialCatalog | null>(null);
  public catalogPage = signal<number>(0);

  public pagedCatalogMaterials = computed(() => {
    const materials = this.catalog()?.status?.availableMaterials ?? [];
    const start = this.catalogPage() * MaterialsComponent.CATALOG_PAGE_SIZE;
    return materials.slice(start, start + MaterialsComponent.CATALOG_PAGE_SIZE);
  });

  public catalogPageCount = computed(() => {
    const total = this.catalog()?.status?.availableMaterials?.length ?? 0;
    return Math.max(1, Math.ceil(total / MaterialsComponent.CATALOG_PAGE_SIZE));
  });
  public refreshing = signal<boolean>(false);
  public submissions = signal<MaterialSubmission[]>([]);
  public skzMaterials = signal<SkzMaterial[]>([]);
  public loading = signal<boolean>(true);
  public showAddDialog = signal<boolean>(false);
  public selectedSourceName = signal<string>('');

  public showRenameDialog = signal<boolean>(false);
  public renameValue = signal<string>('');
  private pendingSource: SkzMaterial | null = null;

  public ngOnInit(): void {
    LuigiClient.addInitListener(() => {
      LuigiClient.uxManager().showLoadingIndicator();
      this.load(() => LuigiClient.uxManager().hideLoadingIndicator());
    });
  }

  private load(onDone?: () => void): void {
    this.client.listCatalogs().subscribe({
      next: (catalogs) => {
        if (catalogs.length > 0) {
          this.catalog.set(catalogs[0]);
          this.finishLoad(onDone);
        } else {
          // First visit: create the singleton, then reload.
          this.client.ensureCatalog().subscribe(() => {
            this.client.listCatalogs().subscribe({
              next: (created) => {
                this.catalog.set(created[0] || null);
                this.finishLoad(onDone);
              },
              error: () => this.finishLoad(onDone),
            });
          });
        }
      },
      error: (err) => {
        console.error('Failed to load material catalog:', err);
        this.finishLoad(onDone);
      },
    });
  }

  private finishLoad(onDone?: () => void): void {
    this.client.listSubmissions().subscribe({
      next: (submissions) => {
        this.submissions.set(submissions);
        this.loading.set(false);
        onDone?.();
      },
      error: (err) => {
        console.error('Failed to load material submissions:', err);
        this.loading.set(false);
        onDone?.();
      },
    });
  }

  /** Requests a StrömungsRaum refresh and polls until the operator has processed it. */
  public refreshCatalog(): void {
    const catalog = this.catalog();
    if (!catalog || this.refreshing()) {
      return;
    }

    const name = catalog.metadata.name;
    const namespace = catalog.metadata.namespace || 'default';
    const requestedAt = new Date().toISOString();

    this.refreshing.set(true);
    this.client.requestCatalogRefresh(name, namespace, requestedAt).subscribe({
      next: (success) => {
        if (success) {
          this.pollForRefresh(name, requestedAt, 20);
        } else {
          this.refreshing.set(false);
          LuigiClient.uxManager().showAlert({
            text: 'Failed to request a refresh',
            type: 'error',
            closeAfter: 3000,
          });
        }
      },
      error: () => {
        this.refreshing.set(false);
        LuigiClient.uxManager().showAlert({
          text: 'Failed to request a refresh',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  /**
   * Polls listCatalogs() every 1.5s until status.lastProcessedRefreshRequestedAt
   * catches up to the timestamp we sent (the reconciler processes a refresh in
   * one reconcile, there's no separate "Refreshing" phase to watch - see
   * operator/materials/materialcatalog_controller.go). Gives up silently after
   * ~30s so a slow/stuck reconcile doesn't disable the button forever.
   */
  private pollForRefresh(name: string, requestedAt: string, attemptsLeft: number): void {
    if (attemptsLeft <= 0) {
      this.refreshing.set(false);
      return;
    }
    setTimeout(() => {
      this.client.listCatalogs().subscribe({
        next: (catalogs) => {
          const updated = catalogs.find((c) => c.metadata.name === name);
          if (updated?.status?.lastProcessedRefreshRequestedAt === requestedAt) {
            this.catalog.set(updated);
            this.catalogPage.set(0);
            this.refreshing.set(false);
          } else {
            this.pollForRefresh(name, requestedAt, attemptsLeft - 1);
          }
        },
        error: () => this.pollForRefresh(name, requestedAt, attemptsLeft - 1),
      });
    }, 1500);
  }

  public prevCatalogPage(): void {
    this.catalogPage.update((p) => Math.max(0, p - 1));
  }

  public nextCatalogPage(): void {
    this.catalogPage.update((p) => Math.min(this.catalogPageCount() - 1, p + 1));
  }

  public formatTimestamp(ts?: string): string {
    return ts ? new Date(ts).toLocaleString() : 'never';
  }

  public openMaterialLink(submission: MaterialSubmission): void {
    const url = submission.status?.materialUrl;
    if (url) {
      window.open(url, '_blank', 'noopener');
    }
  }

  public openAddDialog(): void {
    this.selectedSourceName.set('');
    this.showAddDialog.set(true);
    this.client.listSkzMaterials().subscribe({
      next: (materials) => this.skzMaterials.set(materials),
      error: () => this.skzMaterials.set([]),
    });
  }

  public closeAddDialog(): void {
    this.showAddDialog.set(false);
  }

  public onSourcePick(event: Event): void {
    const select = event.target as HTMLSelectElement & { value: string };
    this.selectedSourceName.set(select.value);
  }

  public onRenameInput(event: Event): void {
    const input = event.target as HTMLInputElement & { value: string };
    this.renameValue.set(input.value);
  }

  public confirmAdd(): void {
    const source = this.skzMaterials().find((m) => m.metadata.name === this.selectedSourceName());
    if (!source || !source.spec) {
      LuigiClient.uxManager().showAlert({
        text: 'Choose a material to submit',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    if (this.nameExistsInBackend(source.spec.name)) {
      // Backend already has a material with this name - ask for a new one
      // instead of letting the operator create a duplicate.
      this.pendingSource = source;
      this.renameValue.set(source.spec.name);
      this.closeAddDialog();
      this.showRenameDialog.set(true);
      return;
    }

    this.submitMaterial(source);
  }

  private nameExistsInBackend(name: string): boolean {
    const existing = this.catalog()?.status?.availableMaterials ?? [];
    return existing.some((m) => m.name === name);
  }

  public confirmRename(): void {
    const source = this.pendingSource;
    if (!source) {
      return;
    }
    const newName = this.renameValue().trim();
    if (!newName) {
      LuigiClient.uxManager().showAlert({ text: 'Enter a name', type: 'warning', closeAfter: 3000 });
      return;
    }
    if (this.nameExistsInBackend(newName)) {
      LuigiClient.uxManager().showAlert({
        text: `"${newName}" already exists too - choose another name`,
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    this.showRenameDialog.set(false);
    this.pendingSource = null;
    this.submitMaterial(source, newName);
  }

  public cancelRename(): void {
    this.showRenameDialog.set(false);
    this.pendingSource = null;
  }

  private submitMaterial(source: SkzMaterial, materialName?: string): void {
    const crName = `${source.metadata.name}-${Date.now().toString(36)}`;

    this.client.createSubmission(crName, source, materialName).subscribe({
      next: (success) => {
        if (success) {
          this.closeAddDialog();
          this.finishLoad();
          this.pollForSubmissionReady(crName, 20);
        } else {
          LuigiClient.uxManager().showAlert({
            text: 'Failed to create submission',
            type: 'error',
            closeAfter: 3000,
          });
        }
      },
      error: () => {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to create submission',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  /**
   * Polls the new MaterialSubmission's own status until the operator has
   * actually submitted it to StrömungsRaum (status.phase 'Ready'), then
   * refreshes the catalog so the new material shows up without a manual
   * click. Submitting to the CRD Gateway only confirms the CR was created,
   * not that the backend call behind it has happened yet.
   */
  private pollForSubmissionReady(crName: string, attemptsLeft: number): void {
    if (attemptsLeft <= 0) {
      return;
    }
    setTimeout(() => {
      this.client.listSubmissions().subscribe({
        next: (subs) => {
          this.submissions.set(subs);
          const sub = subs.find((s) => s.metadata.name === crName);
          if (sub?.status?.phase === 'Ready') {
            this.refreshCatalog();
          } else if (sub?.status?.phase === 'Failed') {
            // Card already shows the error - nothing left to refresh for.
          } else {
            this.pollForSubmissionReady(crName, attemptsLeft - 1);
          }
        },
        error: () => this.pollForSubmissionReady(crName, attemptsLeft - 1),
      });
    }, 1500);
  }

  public deleteSubmission(submission: MaterialSubmission): void {
    LuigiClient.uxManager()
      .showConfirmationModal({
        type: 'warning',
        header: 'Delete Submission',
        body: `Are you sure you want to delete "${submission.metadata.name}"?`,
        buttonConfirm: 'Delete',
        buttonDismiss: 'Cancel',
      })
      .then(() => {
        this.client
          .deleteSubmission(submission.metadata.name, submission.metadata.namespace || 'default')
          .subscribe({
            next: (success) => {
              if (success) {
                this.finishLoad();
              } else {
                LuigiClient.uxManager().showAlert({
                  text: 'Failed to delete submission',
                  type: 'error',
                  closeAfter: 3000,
                });
              }
            },
            error: () => {
              LuigiClient.uxManager().showAlert({
                text: 'Failed to delete submission',
                type: 'error',
                closeAfter: 3000,
              });
            },
          });
      })
      .catch(() => {
        console.log('Deletion cancelled');
      });
  }
}
