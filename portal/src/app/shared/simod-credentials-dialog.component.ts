/**
 * SIMOD Credentials Dialog - shared across every provider page
 *
 * One form, reused by injection-molding.component.ts and
 * materials.component.ts (not duplicated per page): email, password, and
 * backend URL for the shared simod-credentials Secret (see
 * simod-credentials.service.ts). Email and backend URL are prefilled from
 * the existing Secret (if any) every time the dialog opens, so re-opening it
 * shows what's actually configured rather than a blank form. Password stays
 * write-only - never pre-filled, never read back - and is cleared right
 * after every submit attempt either way, so the plaintext never sits in
 * component state longer than the one request it's for.
 */
import { Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, inject, signal } from '@angular/core';
import * as LuigiClient from '@luigi-project/client';
import { ButtonComponent, DialogComponent, InputComponent, LabelComponent, TextComponent } from '@ui5/webcomponents-ngx';

import { DEFAULT_SIMOD_BASE_URL, SimodCredentialsService } from './simod-credentials.service';

@Component({
  selector: 'app-simod-credentials-dialog',
  standalone: true,
  imports: [DialogComponent, InputComponent, LabelComponent, ButtonComponent, TextComponent],
  templateUrl: './simod-credentials-dialog.component.html',
  styleUrl: './simod-credentials-dialog.component.scss',
})
export class SimodCredentialsDialogComponent implements OnChanges {
  @Input() open = false;
  /** true = saved successfully, false = cancelled/dismissed without saving. */
  @Output() closed = new EventEmitter<boolean>();

  private client = inject(SimodCredentialsService);

  public email = signal('');
  public password = signal('');
  public baseUrl = signal(DEFAULT_SIMOD_BASE_URL);
  public saving = signal(false);

  public ngOnChanges(changes: SimpleChanges): void {
    if (changes['open'] && this.open) {
      this.loadExisting();
    }
  }

  private loadExisting(): void {
    this.password.set('');
    this.client.getCredentialInfo().subscribe((info) => {
      this.email.set(info?.email ?? '');
      this.baseUrl.set(info?.baseUrl ?? DEFAULT_SIMOD_BASE_URL);
    });
  }

  public onEmailInput(event: Event): void {
    this.email.set((event.target as HTMLInputElement).value);
  }

  public onPasswordInput(event: Event): void {
    this.password.set((event.target as HTMLInputElement).value);
  }

  public onBaseUrlInput(event: Event): void {
    this.baseUrl.set((event.target as HTMLInputElement).value);
  }

  public cancel(): void {
    this.password.set('');
    this.closed.emit(false);
  }

  public submit(): void {
    const email = this.email().trim();
    const password = this.password();
    const baseUrl = this.baseUrl().trim() || DEFAULT_SIMOD_BASE_URL;
    if (!email || !password) {
      LuigiClient.uxManager().showAlert({
        text: 'Enter both an email and a password',
        type: 'warning',
        closeAfter: 3000,
      });
      return;
    }

    this.saving.set(true);
    this.client.setCredentials(email, password, baseUrl).subscribe((ok) => {
      this.saving.set(false);
      this.password.set('');
      if (ok) {
        LuigiClient.uxManager().showAlert({
          text: 'SIMOD credentials saved',
          type: 'success',
          closeAfter: 3000,
        });
        this.closed.emit(true);
      } else {
        LuigiClient.uxManager().showAlert({
          text: 'Failed to save SIMOD credentials',
          type: 'error',
          closeAfter: 3000,
        });
      }
    });
  }
}
