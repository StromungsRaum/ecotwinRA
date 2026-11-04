/**
 * SKZ Materials Page Component
 *
 * HttpBin-style, same as injection-molding: no cards by default, "+" opens a
 * dialog with a textarea where the user pastes a full material
 * characterization JSON (shape: docs/assets/EastmanPlastics1000.json).
 * Confirming parses it client-side and creates a Material CR with that data
 * verbatim - no canned placeholders anymore (see skz.service.ts). No
 * refresh/monitor/poll - unlike injection-molding, there's no external
 * backend and no operator here, so nothing ever changes after creation.
 */
import { Component, CUSTOM_ELEMENTS_SCHEMA, inject, signal } from '@angular/core';
import * as LuigiClient from '@luigi-project/client';
import {
  ButtonComponent,
  DialogComponent,
  DynamicPageComponent,
  DynamicPageTitleComponent,
  InputComponent,
  LabelComponent,
  TextAreaComponent,
  TextComponent,
  TitleComponent,
  ToolbarButtonComponent,
  ToolbarComponent,
} from '@ui5/webcomponents-ngx';

import '@ui5/webcomponents-icons/dist/add.js';
import '@ui5/webcomponents-icons/dist/decline.js';
import '@ui5/webcomponents-icons/dist/upload.js';

import { Material, MaterialSpec, SkzService } from './skz.service';

/**
 * Top-level keys a pasted MaterialSpec JSON must have (apis/skz/v1alpha1/material_types.go).
 * 'name' is deliberately excluded - it comes from the dialog's own Name field, not the JSON,
 * and gets spliced into the parsed spec in confirmAdd().
 */
const REQUIRED_SPEC_KEYS: (keyof MaterialSpec)[] = [
  'data',
  'viscoModel',
  'heatModel',
  'tempModel',
  'pvtmodel',
  'structure',
  'min_melt_temp',
  'max_melt_temp',
  'abs_max_melt_temp',
  'eject_temp',
  'freeze_temp',
  'melt_density',
  'solid_density',
  'mech_properties',
  'shrinkage_properties',
  'max_shear_stress',
  'max_shear_rate',
];

@Component({
  selector: 'app-skz',
  standalone: true,
  imports: [
    DynamicPageComponent,
    DynamicPageTitleComponent,
    TitleComponent,
    LabelComponent,
    TextComponent,
    ToolbarComponent,
    ToolbarButtonComponent,
    ButtonComponent,
    DialogComponent,
    TextAreaComponent,
    InputComponent,
  ],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './skz.component.html',
  styleUrl: './skz.component.scss',
})
export class SkzComponent {
  private client = inject(SkzService);

  public materials = signal<Material[]>([]);
  public loading = signal<boolean>(true);
  public showAddDialog = signal<boolean>(false);
  public materialName = signal<string>('');
  public materialJson = signal<string>('');
  public jsonError = signal<string>('');

  public ngOnInit(): void {
    LuigiClient.addInitListener(() => {
      LuigiClient.uxManager().showLoadingIndicator();
      this.load(() => LuigiClient.uxManager().hideLoadingIndicator());
    });
  }

  private load(onDone?: () => void): void {
    this.client.list().subscribe({
      next: (materials) => {
        this.materials.set(materials);
        this.loading.set(false);
        onDone?.();
      },
      error: (err) => {
        console.error('Failed to load materials:', err);
        this.loading.set(false);
        onDone?.();
      },
    });
  }

  public openAddDialog(): void {
    this.materialName.set('');
    this.materialJson.set('');
    this.jsonError.set('');
    this.showAddDialog.set(true);
  }

  public closeAddDialog(): void {
    this.showAddDialog.set(false);
  }

  public onNameInput(event: Event): void {
    const input = event.target as HTMLInputElement & { value: string };
    this.materialName.set(input.value);
    this.jsonError.set('');
  }

  public onJsonInput(event: Event): void {
    const textarea = event.target as HTMLTextAreaElement;
    this.setMaterialJson(textarea.value);
  }

  /** Reads a locally-picked JSON file into the textarea, same as pasting it. */
  public onJsonFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    input.value = ''; // allow re-picking the same file later
    if (!file) {
      return;
    }

    const reader = new FileReader();
    reader.onload = () => {
      this.setMaterialJson(typeof reader.result === 'string' ? reader.result : '');
    };
    reader.onerror = () => {
      this.jsonError.set('Failed to read file.');
    };
    reader.readAsText(file);
  }

  private setMaterialJson(value: string): void {
    this.materialJson.set(value);
    this.jsonError.set('');

    // Prefill Name from the JSON's own "name" field, unless the user already typed one.
    if (!this.materialName().trim()) {
      try {
        const parsed = JSON.parse(value) as Record<string, unknown>;
        if (typeof parsed?.['name'] === 'string') {
          this.materialName.set(parsed['name']);
        }
      } catch {
        // still typing / not valid JSON yet - ignore
      }
    }
  }

  /** Parses and shape-checks the pasted JSON; returns null (and sets jsonError) if invalid. */
  private parseMaterialSpec(): MaterialSpec | null {
    const raw = this.materialJson().trim();
    if (!raw) {
      this.jsonError.set('Paste a material JSON (see docs/assets/EastmanPlastics1000.json for the shape).');
      return null;
    }

    let parsed: unknown;
    try {
      parsed = JSON.parse(raw);
    } catch (e) {
      this.jsonError.set(`Invalid JSON: ${(e as Error).message}`);
      return null;
    }

    if (typeof parsed !== 'object' || parsed === null) {
      this.jsonError.set('Expected a JSON object.');
      return null;
    }

    const missing = REQUIRED_SPEC_KEYS.filter((key) => !(key in (parsed as Record<string, unknown>)));
    if (missing.length > 0) {
      this.jsonError.set(`Missing field(s): ${missing.join(', ')}`);
      return null;
    }

    return parsed as MaterialSpec;
  }

  public confirmAdd(): void {
    const name = this.materialName().trim();
    if (!name) {
      this.jsonError.set('Enter a name.');
      return;
    }

    const spec = this.parseMaterialSpec();
    if (!spec) {
      return;
    }
    spec.name = name;

    const crName = spec.name
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '') || 'material';

    this.client.create(crName, spec).subscribe({
      next: (success) => {
        if (success) {
          this.closeAddDialog();
          this.load();
        } else {
          LuigiClient.uxManager().showAlert({
            text: 'Failed to create material',
            type: 'error',
            closeAfter: 3000,
          });
        }
      },
      error: () => {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to create material',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  public deleteMaterial(material: Material): void {
    LuigiClient.uxManager()
      .showConfirmationModal({
        type: 'warning',
        header: 'Delete Material',
        body: `Are you sure you want to delete "${material.spec?.name}"?`,
        buttonConfirm: 'Delete',
        buttonDismiss: 'Cancel',
      })
      .then(() => {
        this.client.delete(material.metadata.name, material.metadata.namespace || 'default').subscribe({
          next: (success) => {
            // Always reload, even on a reported failure: a transient
            // gateway/network hiccup can lose the success response after the
            // underlying delete already went through, which otherwise shows
            // "Failed to delete" for a material that's actually gone. Reloading
            // surfaces the true state instead of trusting a lost response.
            this.load();
            if (!success) {
              LuigiClient.uxManager().showAlert({
                text: 'Failed to delete material',
                type: 'error',
                closeAfter: 3000,
              });
            }
          },
          error: () => {
            this.load();
            LuigiClient.uxManager().showAlert({
              text: 'Failed to delete material',
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
