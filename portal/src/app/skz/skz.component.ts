/**
 * Material Provider A - Materials Page Component (internally still "skz")
 *
 * Read-only catalog: the Materials are static and identical for every
 * consumer, pre-created in the provider workspace and served to each consumer
 * through kcp cached resources (see skz.service.ts). There is nothing to add,
 * upload or delete here, and no refresh/poll - nothing changes at runtime.
 */
import { Component, CUSTOM_ELEMENTS_SCHEMA, inject, signal } from '@angular/core';
import * as LuigiClient from '@luigi-project/client';
import {
  DynamicPageComponent,
  DynamicPageTitleComponent,
  LabelComponent,
  TextComponent,
  TitleComponent,
} from '@ui5/webcomponents-ngx';

import { Material, SkzService } from './skz.service';

@Component({
  selector: 'app-skz',
  standalone: true,
  imports: [DynamicPageComponent, DynamicPageTitleComponent, TitleComponent, LabelComponent, TextComponent],
  schemas: [CUSTOM_ELEMENTS_SCHEMA],
  templateUrl: './skz.component.html',
  styleUrl: './skz.component.scss',
})
export class SkzComponent {
  private client = inject(SkzService);

  public materials = signal<Material[]>([]);
  public loading = signal<boolean>(true);

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
}
