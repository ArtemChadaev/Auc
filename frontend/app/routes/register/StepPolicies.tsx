import { Button } from "~/components/ui/button"
import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle} from "~/components/ui/card"
import {Field, FieldGroup} from "~/components/ui/field"
import {Checkbox} from "~/components/ui/checkbox";
import {Label} from "~/components/ui/label";
import PopupDocsRA from "~/components/popupDocsRA";

const StepPolicies = () => {
  return (
    <Card className="mx-auto max-w-md w-xs">
      <CardHeader>
        <CardTitle>Read and access for our policies</CardTitle>
        <CardDescription>
          Further use of the website without consent is not permitted
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FieldGroup>
          <Field orientation="horizontal">
            <Checkbox id='policies1' />
            <Label htmlFor="policies1" className="gap-0" >Read and access for our
              <PopupDocsRA docs="" name="policies" />
            </Label>
          </Field>
          <Field orientation="horizontal">
            <Checkbox id='policies2' />
            <Label htmlFor="policies2" className="gap-0">Наша политика безопасности
              <PopupDocsRA docs="" name="прочитать" />
            </Label>
          </Field>
        </FieldGroup>
      </CardContent>
      <CardFooter>
        <Field>
          <Button type="submit" className="w-full">
            Verify
          </Button>
        </Field>
      </CardFooter>
    </Card>
  )
}

export default StepPolicies