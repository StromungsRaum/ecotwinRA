import { Component } from "@angular/core";
import { ModelsComponent } from "./models/models.component";

@Component({
  selector: "app-root",
  standalone: true,
  imports: [ModelsComponent],
  template: `<app-models></app-models>`,
  styles: ``,
})
export class AppComponent {}
