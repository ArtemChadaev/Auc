import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle} from "~/components/ui/card";
import {Button} from "~/components/ui/button";
import {type NavigateFunction, redirect} from "react-router";
import {Field, FieldDescription, FieldLabel} from "~/components/ui/field";
import {Input} from "~/components/ui/input";

const StepRegister = () => {
  return (
    <Card className="mx-auto max-w-md w-xs">
      <CardHeader>
        <CardTitle>Write your email</CardTitle>
        <CardDescription>
          A verification code will be sent to your email address.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Field onSubmit={() => {sendCode()}}>
          <div className="flex items-center justify-between">
            <FieldLabel htmlFor="otp-verification">
              Email
            </FieldLabel>
          </div>
          <Input placeholder="user@example.com"/>
          <FieldDescription>
            <a href="/auth">I have account</a>
          </FieldDescription>
        </Field>
      </CardContent>
      <CardFooter>
        <Field>
          <Button className="w-full" type="submit">
            Send code
          </Button>
        </Field>
      </CardFooter>
    </Card>
  )
}

const sendCode = () => {
  //Todo: Сделать отправку на бэк с проверками и т.д. е забыть ошибки выводить
  redirect("/register/confirm");
  return
}
export default StepRegister