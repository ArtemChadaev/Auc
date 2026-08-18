import { Button } from "~/components/ui/button"
import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle} from "~/components/ui/card"
import {Field, FieldDescription, FieldGroup, FieldLabel} from "~/components/ui/field"
import {Checkbox} from "~/components/ui/checkbox";
import {Label} from "~/components/ui/label";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from "~/components/ui/dialog";
import {Input} from "~/components/ui/input";

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
              <Dialog>
            <DialogTrigger render={<Button variant="link" className="p-1">policies1</Button>} />
              <DialogContent className="sm:max-w-sm">
                <DialogHeader>
                  <DialogTitle>Edit profile</DialogTitle>
                  <DialogDescription>
                    Make changes to your profile here. Click save when you&apos;re
                    done.
                  </DialogDescription>
                </DialogHeader>
                <FieldGroup>
                  <Field>
                    <Label htmlFor="name-1">Name</Label>
                    <Input id="name-1" name="name" defaultValue="Pedro Duarte" />
                  </Field>
                  <Field>
                    <Label htmlFor="username-1">Username</Label>
                    <Input id="username-1" name="username" defaultValue="@peduarte" />
                  </Field>
                </FieldGroup>
                <DialogFooter>
                  <DialogClose render={<Button variant="link">Cancel</Button>} />
                  <Button type="submit">Save changes</Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </Label>
          </Field>
          <Field orientation="horizontal">
            <Checkbox id='policies2' />
            <Label htmlFor="policies2" className="gap-0">Read and access for our
              <Dialog>
                <DialogTrigger render={<Button variant="link" className="p-1">policies2</Button>} />
                <DialogContent className="sm:max-w-sm">
                  <DialogHeader>
                    <DialogTitle>Edit profile</DialogTitle>
                    <DialogDescription>
                      Make changes to your profile here. Click save when you&apos;re
                      done.
                    </DialogDescription>
                  </DialogHeader>
                  <FieldGroup>
                    <Field>
                      <Label htmlFor="name-1">Name</Label>
                      <Input id="name-1" name="name" defaultValue="Pedro Duarte" />
                    </Field>
                    <Field>
                      <Label htmlFor="username-1">Username</Label>
                      <Input id="username-1" name="username" defaultValue="@peduarte" />
                    </Field>
                  </FieldGroup>
                  <DialogFooter>
                    <DialogClose render={<Button variant="link">Cancel</Button>} />
                    <Button type="submit">Save changes</Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
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