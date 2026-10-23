/**
 * Injection Molding Service Page Component
 *
 * HttpBin-style: no cards by default, "+" creates one. Each card: a link out
 * to the SIMOD web UI, a Refresh button that fetches the account's
 * Simulations into a dropdown, a Monitor button that starts the operator
 * polling the selected simulation every 5 seconds, and - once complete - a
 * tick plus a results link. The whole list is re-fetched every 5 seconds so
 * the portal reflects the operator's own poll results.
 */
import { Component, CUSTOM_ELEMENTS_SCHEMA, OnDestroy, inject, signal } from '@angular/core';
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
  ToolbarButtonComponent,
  ToolbarComponent,
} from '@ui5/webcomponents-ngx';

import '@ui5/webcomponents-icons/dist/add.js';
import '@ui5/webcomponents-icons/dist/refresh.js';
import '@ui5/webcomponents-icons/dist/action.js';
import '@ui5/webcomponents-icons/dist/accept.js';
import '@ui5/webcomponents-icons/dist/decline.js';

import { InjectionMoldingService, InjectionMoldingServiceClient } from './injection-molding.service';

/** How often the portal re-fetches the list, to reflect the operator's own 5s poll. */
const LIST_POLL_INTERVAL_MS = 5000;

@Component({
  selector: 'app-injection-molding',
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
    SelectComponent,
    OptionComponent,
    DialogComponent,
    InputComponent,
    IconComponent,
  ],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './injection-molding.component.html',
  styleUrl: './injection-molding.component.scss',
})
export class InjectionMoldingComponent implements OnDestroy {
  private client = inject(InjectionMoldingServiceClient);
  private pollHandle?: ReturnType<typeof setInterval>;

  public services = signal<InjectionMoldingService[]>([]);
  public loading = signal<boolean>(true);
  public showAddDialog = signal<boolean>(false);
  public newServiceName = signal<string>('');
  public pendingSelection = signal<Record<string, string>>({});

  public ngOnInit(): void {
    LuigiClient.addInitListener(() => {
      LuigiClient.uxManager().showLoadingIndicator();
      this.load(() => LuigiClient.uxManager().hideLoadingIndicator());
      this.pollHandle = setInterval(() => this.load(), LIST_POLL_INTERVAL_MS);
    });
  }

  public ngOnDestroy(): void {
    if (this.pollHandle) {
      clearInterval(this.pollHandle);
    }
  }

  private load(onDone?: () => void): void {
    this.client.list().subscribe({
      next: (services) => {
        this.services.set(services);
        this.loading.set(false);
        onDone?.();
      },
      error: (err) => {
        console.error('Failed to load injection molding services:', err);
        this.loading.set(false);
        onDone?.();
      },
    });
  }

  public openAddDialog(): void {
    this.newServiceName.set('');
    this.showAddDialog.set(true);
  }

  public closeAddDialog(): void {
    this.showAddDialog.set(false);
  }

  public onNameInput(event: Event): void {
    const input = event.target as HTMLInputElement;
    this.newServiceName.set(input.value);
  }

  public confirmAdd(): void {
    const name = this.newServiceName().trim();
    if (!name) {
      LuigiClient.uxManager().showAlert({
        text: 'Please enter a name',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    this.client.create(name, 'default').subscribe({
      next: (success) => {
        if (success) {
          this.closeAddDialog();
          this.load();
        } else {
          LuigiClient.uxManager().showAlert({
            text: 'Failed to create injection molding service',
            type: 'error',
            closeAfter: 3000,
          });
        }
      },
      error: () => {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to create injection molding service',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  public openSimodLink(svc: InjectionMoldingService): void {
    const url = svc.status?.url;
    if (url) {
      window.open(url, '_blank', 'noopener');
    }
  }

  public refresh(svc: InjectionMoldingService): void {
    this.client.requestRefresh(svc.metadata.name, svc.metadata.namespace || 'default').subscribe({
      next: (ok) => {
        if (!ok) {
          LuigiClient.uxManager().showAlert({
            text: 'Failed to request a refresh',
            type: 'error',
            closeAfter: 3000,
          });
          return;
        }
        // The operator's refresh is near-instant (a single list call); a
        // short delay then a reload is enough to pick it up, same idea as
        // the continuous 5s poll but immediate for a user-triggered refresh.
        setTimeout(() => this.load(), 1500);
      },
      error: () => {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to request a refresh',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }

  public onSimulationPick(svc: InjectionMoldingService, event: Event): void {
    const select = event.target as any;
    this.pendingSelection.update((map) => ({ ...map, [svc.metadata.name]: select.selectedOption?.value || '' }));
  }

  public deleteService(svc: InjectionMoldingService): void {
    LuigiClient.uxManager()
      .showConfirmationModal({
        type: 'warning',
        header: 'Delete Injection Molding Service',
        body: `Are you sure you want to delete "${svc.metadata.name}"?`,
        buttonConfirm: 'Delete',
        buttonDismiss: 'Cancel',
      })
      .then(() => {
        this.client.delete(svc.metadata.name, svc.metadata.namespace || 'default').subscribe({
          next: (success) => {
            if (success) {
              this.load();
            } else {
              LuigiClient.uxManager().showAlert({
                text: 'Failed to delete injection molding service',
                type: 'error',
                closeAfter: 3000,
              });
            }
          },
          error: () => {
            LuigiClient.uxManager().showAlert({
              text: 'Failed to delete injection molding service',
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

  public formatTimestamp(timestamp: string | undefined): string {
    if (!timestamp) {
      return 'Never';
    }
    try {
      return new Date(timestamp).toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
      });
    } catch {
      return timestamp;
    }
  }

  public monitor(svc: InjectionMoldingService): void {
    const simulationId = this.pendingSelection()[svc.metadata.name];
    if (!simulationId) {
      LuigiClient.uxManager().showAlert({
        text: 'Please select a simulation first',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    this.client.selectSimulation(svc.metadata.name, svc.metadata.namespace || 'default', simulationId).subscribe({
      next: (ok) => {
        if (!ok) {
          LuigiClient.uxManager().showAlert({
            text: 'Failed to start monitoring the simulation',
            type: 'error',
            closeAfter: 3000,
          });
          return;
        }
        setTimeout(() => this.load(), 1500);
      },
      error: () => {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to start monitoring the simulation',
          type: 'error',
          closeAfter: 3000,
        });
      },
    });
  }
}
