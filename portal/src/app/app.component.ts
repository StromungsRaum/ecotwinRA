import { Component, HostListener, signal } from "@angular/core";
import { InjectionMoldingComponent } from "./injection-molding/injection-molding.component";
import { SkzComponent } from "./skz/skz.component";
import { MaterialsComponent } from "./materials/materials.component";

// Each page is deployed under its own Gateway path prefix (/injection-molding,
// /generic-materials, /materials) with a relative <base href="./"> so assets resolve under
// whichever prefix loaded the page (see src/index.html) - that also means
// Angular's Router sees zero remaining path segments regardless of which
// prefix was used (the base already "consumes" the whole prefix), so page
// selection can't go through Router path matching. Pick the component
// directly from the actual browser path instead.
//
// The path has to be re-read on every Luigi navigation, not just at startup:
// all three pages are served from the same origin, and Luigi reuses the
// existing iframe when the next node's viewUrl has the same origin as the
// current one. Instead of loading the new URL it sends the micro frontend a
// "luigi.navigate" message, on which LuigiClient does
// history.replaceState(viewUrl) and dispatches a "popstate" event - i.e. it
// expects the app to route client-side. Without the popstate listener below,
// clicking from one provider's nav entry to another's kept showing the first
// page until a full reload.
@Component({
  selector: "app-root",
  standalone: true,
  imports: [InjectionMoldingComponent, SkzComponent, MaterialsComponent],
  template: `
    @if (page() === "generic-materials") {
      <app-skz></app-skz>
    } @else if (page() === "materials") {
      <app-materials></app-materials>
    } @else {
      <app-injection-molding></app-injection-molding>
    }
  `,
  styles: ``,
})
export class AppComponent {
  protected readonly page = signal(currentPage());

  @HostListener("window:popstate")
  protected onLocationChange(): void {
    this.page.set(currentPage());
  }
}

type Page = "generic-materials" | "materials" | "injection-molding";

function currentPage(): Page {
  const path = window.location.pathname;
  // Checked before "/materials" on purpose, though neither contains the other.
  if (path.includes("/generic-materials")) {
    return "generic-materials";
  }
  if (path.includes("/materials")) {
    return "materials";
  }
  return "injection-molding";
}
