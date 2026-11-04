import { Component } from "@angular/core";
import { InjectionMoldingComponent } from "./injection-molding/injection-molding.component";
import { SkzComponent } from "./skz/skz.component";
import { MaterialsComponent } from "./materials/materials.component";

// Each page is deployed under its own Gateway path prefix (/injection-molding,
// /skz, /materials) with a relative <base href="./"> so assets resolve under
// whichever prefix loaded the page (see src/index.html) - that also means
// Angular's Router sees zero remaining path segments regardless of which
// prefix was used (the base already "consumes" the whole prefix), so page
// selection can't go through Router path matching. Pick the component
// directly from the actual browser path instead.
@Component({
  selector: "app-root",
  standalone: true,
  imports: [InjectionMoldingComponent, SkzComponent, MaterialsComponent],
  template: `
    @if (isSkz) {
      <app-skz></app-skz>
    } @else if (isMaterials) {
      <app-materials></app-materials>
    } @else {
      <app-injection-molding></app-injection-molding>
    }
  `,
  styles: ``,
})
export class AppComponent {
  protected readonly isSkz = window.location.pathname.includes("/skz");
  protected readonly isMaterials = window.location.pathname.includes("/materials");
}
