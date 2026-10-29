/**
 * SKZ Materials Page Component
 *
 * HttpBin-style, same as injection-molding: no cards by default, "+" opens a
 * dialog with a single Name field (e.g. "Luran 368R"). Confirming creates a
 * Material CR with that name and canned placeholder parameter
 * values (see skz.service.ts). No refresh/monitor/poll - unlike
 * injection-molding, there's no external backend and no operator here, so
 * nothing ever changes after creation.
 */
import { Component, CUSTOM_ELEMENTS_SCHEMA, inject, signal } from '@angular/core';
import { KeyValuePipe } from '@angular/common';
import * as LuigiClient from '@luigi-project/client';
import {
  ButtonComponent,
  DialogComponent,
  DynamicPageComponent,
  DynamicPageTitleComponent,
  InputComponent,
  LabelComponent,
  TextComponent,
  TitleComponent,
  ToolbarButtonComponent,
  ToolbarComponent,
} from '@ui5/webcomponents-ngx';

import '@ui5/webcomponents-icons/dist/add.js';
import '@ui5/webcomponents-icons/dist/decline.js';

import { Material, SkzService } from './skz.service';

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
    InputComponent,
    KeyValuePipe,
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
  public newMaterialName = signal<string>('');

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
        console.error('Failed to load material parameters:', err);
        this.loading.set(false);
        onDone?.();
      },
    });
  }

  public openAddDialog(): void {
    this.newMaterialName.set('');
    this.showAddDialog.set(true);
  }

  public closeAddDialog(): void {
    this.showAddDialog.set(false);
  }

  public onNameInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.newMaterialName.set(input.value);
  }

  public confirmAdd(): void {
    const materialName = this.newMaterialName().trim();
    if (!materialName) {
      LuigiClient.uxManager().showAlert({
        text: 'Please enter a material name',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    const crName = materialName
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '') || 'material';

    this.client.create(crName, 'default', materialName).subscribe({
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
            if (success) {
              this.load();
            } else {
              LuigiClient.uxManager().showAlert({
                text: 'Failed to delete material',
                type: 'error',
                closeAfter: 3000,
              });
            }
          },
          error: () => {
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
