import { Component } from "@angular/core";
import { ModelsComponent } from "./models/models.component";
import { InjectionMoldingComponent } from "./injection-molding/injection-molding.component";

// Each page is deployed under its own Gateway path prefix (/models,
// /injection-molding) with a relative <base href="./"> so assets resolve
// under whichever prefix loaded the page (see src/index.html) - that also
// means Angular's Router sees zero remaining path segments regardless of
// which prefix was used (the base already "consumes" the whole prefix), so
// page selection can't go through Router path matching. Pick the component
// directly from the actual browser path instead.
@Component({
  selector: "app-root",
  standalone: true,
  imports: [ModelsComponent, InjectionMoldingComponent],
  template: `
    @if (isInjectionMolding) {
      <app-injection-molding></app-injection-molding>
    } @else {
      <app-models></app-models>
    }
  `,
  styles: ``,
})
export class AppComponent {
  protected readonly isInjectionMolding = window.location.pathname.includes("injection-molding");
}
