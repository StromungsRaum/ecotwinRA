/**
 * Models Management Page Component
 *
 * This is an example microfrontend that demonstrates how to:
 * 1. Integrate with the Luigi shell via @luigi-project/client
 * 2. Access the Portal context (auth token, API URLs) via LuigiContextService
 * 3. Make GraphQL API calls to manage Kubernetes custom resources
 * 4. Use SAP UI5 web components for a consistent look and feel
 *
 * Key concepts:
 * - LuigiClient.addInitListener() - Wait for Luigi shell handshake before loading data
 * - LuigiContextService - Angular service that provides the portal context as an Observable
 * - context.token - Bearer token for API authentication
 * - context.portalContext.crdGatewayApiUrl - GraphQL endpoint for K8s resources
 */
import { Component, CUSTOM_ELEMENTS_SCHEMA, effect, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import * as LuigiClient from '@luigi-project/client';
import { ILuigiContextTypes, LuigiContextService } from '@luigi-project/client-support-angular';
import {
  AvatarComponent,
  ButtonComponent,
  DialogComponent,
  DynamicPageComponent,
  DynamicPageHeaderComponent,
  DynamicPageTitleComponent,
  IconComponent,
  InputComponent,
  LabelComponent,
  OptionComponent,
  SelectComponent,
  TextComponent,
  TitleComponent,
  ToolbarButtonComponent,
  ToolbarComponent,
} from '@ui5/webcomponents-ngx';

// Import UI5 icons used in the template
import '@ui5/webcomponents-icons/dist/add.js';
import '@ui5/webcomponents-icons/dist/calendar.js';
import '@ui5/webcomponents-icons/dist/delete.js';
import '@ui5/webcomponents-icons/dist/person-placeholder.js';
import '@ui5/webcomponents-icons/dist/refresh.js';

import { Model, ModelsService, Namespace } from './models.service';

@Component({
  selector: 'app-models',
  standalone: true,
  imports: [
    DynamicPageComponent,
    DynamicPageTitleComponent,
    DynamicPageHeaderComponent,
    AvatarComponent,
    TitleComponent,
    LabelComponent,
    TextComponent,
    ToolbarComponent,
    ToolbarButtonComponent,
    IconComponent,
    InputComponent,
    ButtonComponent,
    DialogComponent,
    SelectComponent,
    OptionComponent,
  ],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './models.component.html',
  styleUrl: './models.component.scss',
})
export class ModelsComponent {
  private luigiContextService = inject(LuigiContextService);
  private modelsService = inject(ModelsService);

  public luigiContext = toSignal(this.luigiContextService.contextObservable(), {
    initialValue: { context: {}, contextType: ILuigiContextTypes.INIT },
  });

  public models = signal<Model[]>([]);
  public namespaces = signal<Namespace[]>([]);
  public loading = signal<boolean>(true);
  public showAddDialog = signal<boolean>(false);
  public newModelName = signal<string>('');
  public newModelNamespace = signal<string>('');
  public newModelIntent = signal<string>('');

  constructor() {
    // Debug: Log the full Luigi context whenever it changes
    effect(() => {
      const ctx = this.luigiContext();
      console.log('=== LUIGI CONTEXT ===');
      console.log('Full context:', JSON.stringify(ctx.context, null, 2));
      console.log('Context type:', ctx.contextType);
      console.log('=====================');
    });
  }

  /**
   * Initialize the component after Luigi shell handshake completes.
   *
   * IMPORTANT: Always wait for LuigiClient.addInitListener() before making API calls.
   * This ensures the context (auth token, API URLs) is available.
   */
  public ngOnInit(): void {
    LuigiClient.addInitListener(() => {
      // Show Luigi's loading indicator while fetching data
      LuigiClient.uxManager().showLoadingIndicator();
      this.loadNamespaces();
      this.loadModels();
    });
  }

  public loadNamespaces(): void {
    this.modelsService.listNamespaces().subscribe({
      next: (namespaces) => {
        this.namespaces.set(namespaces);
        // Pre-select first namespace if available
        if (namespaces.length > 0 && !this.newModelNamespace()) {
          this.newModelNamespace.set(namespaces[0].metadata.name);
        }
      },
      error: (err) => {
        console.error('Failed to load namespaces:', err);
      },
    });
  }

  public loadModels(): void {
    this.loading.set(true);
    this.modelsService.listModels().subscribe({
      next: (models) => {
        this.models.set(models);
        this.loading.set(false);
        LuigiClient.uxManager().hideLoadingIndicator();
      },
      error: (err) => {
        console.error('Failed to load models:', err);
        this.loading.set(false);
        LuigiClient.uxManager().hideLoadingIndicator();
        LuigiClient.uxManager().showAlert({
          text: 'Failed to load models',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  public openAddDialog(): void {
    this.newModelName.set('');
    this.newModelIntent.set('');
    // Pre-select first namespace if available
    if (this.namespaces().length > 0) {
      this.newModelNamespace.set(this.namespaces()[0].metadata.name);
    }
    this.showAddDialog.set(true);
  }

  public closeAddDialog(): void {
    this.showAddDialog.set(false);
  }

  public onNameInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.newModelName.set(input.value);
  }

  public onIntentInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.newModelIntent.set(input.value);
  }

  public onNamespaceChange(event: Event): void {
    const select = event.target as any;
    this.newModelNamespace.set(select.selectedOption?.value || '');
  }

  public confirmAddModel(): void {
    const name = this.newModelName().trim();
    const namespace = this.newModelNamespace().trim();
    const intent = this.newModelIntent().trim();

    if (!name) {
      LuigiClient.uxManager().showAlert({
        text: 'Please enter a name for the model',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    if (!namespace) {
      LuigiClient.uxManager().showAlert({
        text: 'Please select a namespace',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    this.modelsService.createModel(name, namespace, intent || undefined).subscribe({
      next: (success) => {
        if (success) {
          LuigiClient.uxManager().showAlert({
            text: `Model "${name}" created successfully`,
            type: 'success',
            closeAfter: 3000,
          });
          this.closeAddDialog();
          this.loadModels();
        } else {
          LuigiClient.uxManager().showAlert({
            text: 'Failed to create model',
            type: 'error',
            closeAfter: 3000,
          });
        }
      },
      error: () => {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to create model',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  public deleteModel(model: Model): void {
    LuigiClient.uxManager()
      .showConfirmationModal({
        type: 'warning',
        header: 'Delete Model',
        body: `Are you sure you want to delete "${model.metadata.name}"?`,
        buttonConfirm: 'Delete',
        buttonDismiss: 'Cancel',
      })
      .then(() => {
        this.modelsService.deleteModel(model.metadata.name, model.metadata.namespace!).subscribe({
          next: (success) => {
            if (success) {
              LuigiClient.uxManager().showAlert({
                text: `Model "${model.metadata.name}" deleted`,
                type: 'success',
                closeAfter: 3000,
              });
              this.loadModels();
            } else {
              LuigiClient.uxManager().showAlert({
                text: 'Failed to delete model',
                type: 'error',
                closeAfter: 3000,
              });
            }
          },
          error: () => {
            LuigiClient.uxManager().showAlert({
              text: 'Failed to delete model',
              type: 'error',
              closeAfter: 3000,
            });
          },
        });
      })
      .catch(() => {
        console.log('Model deletion cancelled');
      });
  }

  public getInitials(name: string): string {
    if (!name) return '??';
    const parts = name.split(/[-_\s]+/);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return name.substring(0, 2).toUpperCase();
  }

  public getColorScheme(name: string): 'Accent1' | 'Accent2' | 'Accent3' | 'Accent4' | 'Accent5' | 'Accent6' | 'Accent7' | 'Accent8' | 'Accent9' | 'Accent10' {
    const schemes = ['Accent1', 'Accent2', 'Accent3', 'Accent4', 'Accent5', 'Accent6', 'Accent7', 'Accent8', 'Accent9', 'Accent10'] as const;
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    return schemes[Math.abs(hash) % schemes.length];
  }

  public formatDate(timestamp: string | undefined): string {
    if (!timestamp) return 'Unknown';
    try {
      const date = new Date(timestamp);
      return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    } catch {
      return timestamp;
    }
  }
}
